package http

import (
	"mkk_basis/rest_api/api/docs"
	auth_router "mkk_basis/rest_api/internal/app/core/transport/rest/auth-router"
	tasks_router "mkk_basis/rest_api/internal/app/core/transport/rest/tasks-router"
	teams_router "mkk_basis/rest_api/internal/app/core/transport/rest/teams-router"
	users_router "mkk_basis/rest_api/internal/app/core/transport/rest/users-router"
	middlewares "mkk_basis/rest_api/internal/components/http/middlewares"
	"mkk_basis/rest_api/internal/config"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
)

func GetRoutes() *gin.Engine {
	gin.SetMode(gin.DebugMode)
	router := gin.New()
	router.ForwardedByClientIP = false
	err := router.SetTrustedProxies(nil)
	if err != nil {
		return nil
	}

	// Logger middleware
	skipPaths := map[string]bool{}
	router.Use(func(c *gin.Context) {
		if skipPaths[c.Request.URL.Path] {
			c.Next()
			return
		}
		ginzap.Ginzap(serverLogger, time.RFC3339, true)(c)
	})
	router.Use(ginzap.RecoveryWithZap(serverLogger, true))
	router.Use(middlewares.MetricsMiddleware())
	router.Use(cors.New(cors.Config{
		AllowOriginFunc: func(origin string) bool {
			return true
		},
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders: []string{
			"Origin",
			"Content-Type",
			"Authorization",
			"X-Requested-With",
		},
		ExposeHeaders: []string{
			"Content-Type",
			"Content-Range",
		},
		AllowCredentials:          true,
		MaxAge:                    12 * time.Hour,
		OptionsResponseStatusCode: 204,
	}))

	// API routes
	AddApiRoutes(router)
	metricsPath := "/metrics"
	if config.CurrentConfig.Metrics != nil && config.CurrentConfig.Metrics.Path != "" {
		metricsPath = config.CurrentConfig.Metrics.Path
	}
	router.GET(metricsPath, middlewares.MetricsHandler())
	router.Group("swagger").
		GET("/*any", func(c *gin.Context) {
			docs.SwaggerInfo.Host = c.Request.Host
		}, ginSwagger.WrapHandler(swaggerFiles.Handler))

	// 404 handler
	router.NoRoute(func(c *gin.Context) {
		serverLogger.Warn("Route not found",
			zap.String("path", c.Request.URL.Path),
			zap.String("method", c.Request.Method))

		c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
	})

	return router
}

// AddApiRoutes adds all API routes to the router
func AddApiRoutes(router *gin.Engine) {

	api := router.Group("/api/v1")
	auth_router.AddRoutes(api)
	users_router.AddRoutes(api)
	teams_router.AddRoutes(api)
	tasks_router.AddRoutes(api)
}
