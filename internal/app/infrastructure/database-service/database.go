package database_service

import (
	"fmt"
	"mkk_basis/rest_api/internal/config"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type dbConn struct {
	DB *gorm.DB
}

func NewConn() *dbConn {
	return &dbConn{}
}

func (c *dbConn) Connect() error {
	dbConf := config.CurrentConfig.Database
	dbLogger.Infof("connect to mysql; host=%s port=%s db=%s user=%s",
		dbConf.DbHost,
		dbConf.DbPort,
		dbConf.DbName,
		dbConf.DbUser,
	)
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true&loc=UTC&timeout=5s&readTimeout=5s&writeTimeout=5s&multiStatements=true",
		dbConf.DbUser,
		dbConf.DbPassword,
		dbConf.DbHost,
		dbConf.DbPort,
		dbConf.DbName,
	)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		return fmt.Errorf("failed to open mysql connection: %w", err)
	}

	if err := c.configureConnectionPool(db, dbConf); err != nil {
		return err
	}

	c.DB = db

	dbLogger.Info("mysql connected successfully")

	return nil
}

func (c *dbConn) configureConnectionPool(db *gorm.DB, dbConf *config.DataBaseConfig) error {
	dbLogger.Info("configure mysql connection pool")

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql db: %w", err)
	}

	sqlDB.SetMaxOpenConns(dbConf.MaxConnections)
	sqlDB.SetMaxIdleConns(dbConf.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(dbConf.ConnMaxLifetimeMinutes) * time.Minute)
	sqlDB.SetConnMaxIdleTime(time.Duration(dbConf.ConnMaxIdleTimeMinutes) * time.Minute)

	dbLogger.Infof(
		"mysql pool configured",
	)
	return nil
}

func (c *dbConn) Stop() error {
	if c.DB == nil {
		dbLogger.Info("mysql connection is nil; nothing to close")
		return nil
	}

	sqlDB, err := c.DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql db: %w", err)
	}

	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("failed to close mysql connection: %w", err)
	}

	dbLogger.Info("mysql connection closed")

	return nil
}
