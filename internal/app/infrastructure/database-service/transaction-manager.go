package database_service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"mkk_basis/rest_api/internal/migrations"
	"sync"
	"sync/atomic"

	"gorm.io/gorm"
)

type transactionContextKey struct{}
type TransactionManager interface {
	IsInTransaction(ctx context.Context) bool
	DBRun(
		ctx context.Context,
		fn func(ctx context.Context, tx *gorm.DB) error,
		opts ...*sql.TxOptions,
	) error
	DBRunWithoutTx(fn func(tx *gorm.DB) error) error
	Launch() error
	Update() error
	Stop() error
	Migration() error
	SetConnection(conn *gorm.DB)
	Recover() error
	RecoverMigration() error
}

type TransactionManagerImpl struct {
	dbConn  *dbConn
	wg      sync.WaitGroup
	stopped atomic.Bool
}

func NewTransactionManager() TransactionManager {
	return &TransactionManagerImpl{wg: sync.WaitGroup{}}
}

func (tm *TransactionManagerImpl) Launch() error {
	tm.dbConn = NewConn()

	if err := tm.dbConn.Connect(); err != nil {
		return err
	}
	return nil
}

func (tm *TransactionManagerImpl) Stop() error {
	tm.stopped.Store(true)
	tm.wg.Wait()
	if err := tm.dbConn.Stop(); err != nil {
		dbLogger.Errorf("Error while stopping DB connect")
		return err
	}
	tm.dbConn = nil
	return nil
}

func (tm *TransactionManagerImpl) Update() error {
	if tm.dbConn != nil {
		if err := tm.dbConn.Stop(); err != nil {
			dbLogger.Errorf("Error while stopping DB connect")
			return err
		}
	}
	newDBConn := NewConn()
	if err := newDBConn.Connect(); err != nil {
		return err
	}
	tm.dbConn = newDBConn
	return nil
}

func (tm *TransactionManagerImpl) getDB(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(transactionContextKey{}).(*gorm.DB); ok && tx != nil {
		return tx
	}
	return tm.dbConn.DB
}

func (tm *TransactionManagerImpl) IsInTransaction(ctx context.Context) bool {
	_, ok := ctx.Value(transactionContextKey{}).(*gorm.DB)
	return ok
}

func (tm *TransactionManagerImpl) DBRun(
	ctx context.Context,
	fn func(ctx context.Context, tx *gorm.DB) error,
	opts ...*sql.TxOptions,
) error {
	tm.wg.Add(1)
	defer tm.wg.Done()

	if tm.IsInTransaction(ctx) {
		return fn(ctx, tm.getDB(ctx))
	}

	if isStop := tm.stopped.Load(); isStop {
		return errors.New("transaction manager is stopped")
	}

	return tm.dbConn.DB.Transaction(func(tx *gorm.DB) error {
		ctxWithTx := context.WithValue(ctx, transactionContextKey{}, tx)
		return fn(ctxWithTx, tx)
	}, opts...)
}

func (tm *TransactionManagerImpl) DBRunWithoutTx(fn func(tx *gorm.DB) error) error {
	return fn(tm.dbConn.DB)
}

func (tm *TransactionManagerImpl) Migration() (err error) {
	sqlDb, err := tm.dbConn.DB.DB()
	if err != nil {
		return err
	}
	if err := migrations.RunMigrations(sqlDb); err != nil {
		return err
	}
	return
}

func (tm *TransactionManagerImpl) SetConnection(conn *gorm.DB) {
	tm.dbConn = NewConn()
	tm.dbConn.DB = conn
}

func (tm *TransactionManagerImpl) Recover() error {
	if err := tm.Launch(); err != nil {
		return fmt.Errorf("cant connect db: %w", err)
	}
	if err := tm.RecoverMigration(); err != nil {
		return fmt.Errorf("cant run migrations: %w", err)
	}
	return nil
}

func (tm *TransactionManagerImpl) RecoverMigration() error {
	if err := tm.Migration(); err != nil {
		return fmt.Errorf("cant run migrations: %w", err)
	}
	return nil
}
