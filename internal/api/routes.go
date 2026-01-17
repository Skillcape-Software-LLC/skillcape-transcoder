package api

import (
	"github.com/gin-gonic/gin"
	"github.com/skillcape/transcoder/internal/config"
	"github.com/skillcape/transcoder/internal/jobs"
	"github.com/skillcape/transcoder/internal/storage"
)

func SetupRouter(cfg *config.Config, localStorage *storage.LocalStorage, jobQueue *jobs.Queue) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()

	// Global middleware
	router.Use(Recovery())
	router.Use(RequestLogger())
	router.Use(CORS())

	// Create handler
	handler := NewHandler(localStorage, jobQueue)

	// Health check (no auth required)
	router.GET("/health", handler.HealthCheck)

	// API v1 routes (auth required)
	v1 := router.Group("/api/v1")
	v1.Use(APIKeyAuth(cfg.APIKey))
	{
		v1.POST("/jobs", handler.CreateJob)
		v1.GET("/jobs", handler.ListJobs)
		v1.GET("/jobs/:id", handler.GetJob)
		v1.DELETE("/jobs/:id", handler.DeleteJob)
	}

	return router
}
