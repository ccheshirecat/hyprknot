package api

import (
	"github.com/gin-gonic/gin"
	"github.com/hypr-technologies/hyprknot/internal/config"
	"github.com/hypr-technologies/hyprknot/internal/knot"
	"github.com/sirupsen/logrus"
)

// SetupRoutes sets up all API routes
func SetupRoutes(cfg *config.Config, knotClient *knot.Client, logger *logrus.Logger) *gin.Engine {
	// Set Gin mode based on log level
	if cfg.Log.Level == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Create handler
	handler := NewHandler(knotClient, logger)

	// Global middleware
	router.Use(ErrorHandlingMiddleware(logger))
	router.Use(LoggingMiddleware(logger))
	router.Use(SecurityHeadersMiddleware())
	router.Use(CORSMiddleware())
	router.Use(RequestIDMiddleware())
	router.Use(RateLimitMiddleware())

	// Health check endpoint (no auth required)
	router.GET("/health", handler.HealthCheck)

	// API routes with authentication
	api := router.Group("/api/v1")
	api.Use(AuthMiddleware(cfg.Auth.APIKeys, cfg.Auth.Enabled))

	// Zone routes
	api.GET("/zones", handler.GetZones)
	api.POST("/zones/:zone/reload", handler.ReloadZone)

	// Record routes
	api.GET("/zones/:zone/records", handler.GetRecords)
	api.GET("/zones/:zone/records/:name/:type", handler.GetRecord)
	api.POST("/zones/:zone/records", handler.CreateRecord)
	api.PUT("/zones/:zone/records/:name/:type", handler.UpdateRecord)
	api.DELETE("/zones/:zone/records/:name/:type", handler.DeleteRecord)

	// API documentation endpoint
	api.GET("/docs", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"service": "HyprKnot DNS API",
			"version": "1.0.0",
			"endpoints": map[string]interface{}{
				"health": map[string]string{
					"method": "GET",
					"path":   "/health",
					"desc":   "Health check endpoint",
				},
				"zones": map[string]interface{}{
					"list": map[string]string{
						"method": "GET",
						"path":   "/api/v1/zones",
						"desc":   "List all zones",
					},
					"reload": map[string]string{
						"method": "POST",
						"path":   "/api/v1/zones/{zone}/reload",
						"desc":   "Reload a zone",
					},
				},
				"records": map[string]interface{}{
					"list": map[string]string{
						"method": "GET",
						"path":   "/api/v1/zones/{zone}/records",
						"desc":   "List all records in a zone",
					},
					"get": map[string]string{
						"method": "GET",
						"path":   "/api/v1/zones/{zone}/records/{name}/{type}",
						"desc":   "Get a specific record",
					},
					"create": map[string]string{
						"method": "POST",
						"path":   "/api/v1/zones/{zone}/records",
						"desc":   "Create a new record",
					},
					"update": map[string]string{
						"method": "PUT",
						"path":   "/api/v1/zones/{zone}/records/{name}/{type}",
						"desc":   "Update an existing record",
					},
					"delete": map[string]string{
						"method": "DELETE",
						"path":   "/api/v1/zones/{zone}/records/{name}/{type}",
						"desc":   "Delete a record",
					},
				},
			},
			"supported_record_types": []string{
				"A", "AAAA", "PTR", "CNAME", "MX", "TXT", "NS",
			},
			"authentication": map[string]interface{}{
				"enabled": cfg.Auth.Enabled,
				"method":  "API Key",
				"headers": []string{"X-API-Key", "Authorization: Bearer <token>"},
			},
		})
	})

	return router
}
