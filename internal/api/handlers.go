package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hypr-technologies/hyprknot/internal/knot"
	"github.com/sirupsen/logrus"
)

// Handler represents the API handler
type Handler struct {
	knotClient *knot.Client
	logger     *logrus.Logger
}

// NewHandler creates a new API handler
func NewHandler(knotClient *knot.Client, logger *logrus.Logger) *Handler {
	return &Handler{
		knotClient: knotClient,
		logger:     logger,
	}
}

// HealthCheck handles health check requests
func (h *Handler) HealthCheck(c *gin.Context) {
	// Check KnotDNS health
	if err := h.knotClient.CheckHealth(); err != nil {
		h.logger.Errorf("Health check failed: %v", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unhealthy",
			"error":  "KnotDNS is not accessible",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "hyprknot",
		"version": "1.0.0",
	})
}

// GetZones handles GET /api/v1/zones
func (h *Handler) GetZones(c *gin.Context) {
	zones, err := h.knotClient.GetZones()
	if err != nil {
		h.logger.Errorf("Failed to get zones: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve zones",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"zones": zones,
	})
}

// GetRecords handles GET /api/v1/zones/:zone/records
func (h *Handler) GetRecords(c *gin.Context) {
	zone := c.Param("zone")
	if zone == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Zone parameter is required",
		})
		return
	}

	records, err := h.knotClient.GetRecords(zone)
	if err != nil {
		h.logger.Errorf("Failed to get records for zone %s: %v", zone, err)
		if strings.Contains(err.Error(), "zone not allowed") {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Access to zone not allowed",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve records",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"zone":    zone,
		"records": records,
	})
}

// GetRecord handles GET /api/v1/zones/:zone/records/:name/:type
func (h *Handler) GetRecord(c *gin.Context) {
	zone := c.Param("zone")
	name := c.Param("name")
	recordType := knot.RecordType(strings.ToUpper(c.Param("type")))

	if zone == "" || name == "" || recordType == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Zone, name, and type parameters are required",
		})
		return
	}

	if !knot.IsValidRecordType(string(recordType)) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid record type",
		})
		return
	}

	record, err := h.knotClient.GetRecord(zone, name, recordType)
	if err != nil {
		h.logger.Errorf("Failed to get record %s %s in zone %s: %v", name, recordType, zone, err)
		if strings.Contains(err.Error(), "zone not allowed") {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Access to zone not allowed",
			})
			return
		}
		if strings.Contains(err.Error(), "record not found") {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Record not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve record",
		})
		return
	}

	c.JSON(http.StatusOK, record)
}

// CreateRecord handles POST /api/v1/zones/:zone/records
func (h *Handler) CreateRecord(c *gin.Context) {
	zone := c.Param("zone")
	if zone == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Zone parameter is required",
		})
		return
	}

	var req knot.CreateRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	if err := req.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	record := req.ToRecord()
	if err := h.knotClient.CreateRecord(zone, record); err != nil {
		h.logger.Errorf("Failed to create record in zone %s: %v", zone, err)
		if strings.Contains(err.Error(), "zone not allowed") {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Access to zone not allowed",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create record",
		})
		return
	}

	h.logger.Infof("Created record %s %s in zone %s", record.Name, record.Type, zone)
	c.JSON(http.StatusCreated, record)
}

// UpdateRecord handles PUT /api/v1/zones/:zone/records/:name/:type
func (h *Handler) UpdateRecord(c *gin.Context) {
	zone := c.Param("zone")
	name := c.Param("name")
	recordType := knot.RecordType(strings.ToUpper(c.Param("type")))

	if zone == "" || name == "" || recordType == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Zone, name, and type parameters are required",
		})
		return
	}

	if !knot.IsValidRecordType(string(recordType)) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid record type",
		})
		return
	}

	var req knot.UpdateRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	if err := h.knotClient.UpdateRecord(zone, name, recordType, &req); err != nil {
		h.logger.Errorf("Failed to update record %s %s in zone %s: %v", name, recordType, zone, err)
		if strings.Contains(err.Error(), "zone not allowed") {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Access to zone not allowed",
			})
			return
		}
		if strings.Contains(err.Error(), "record not found") {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Record not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update record",
		})
		return
	}

	// Get updated record to return
	updatedRecord, err := h.knotClient.GetRecord(zone, name, recordType)
	if err != nil {
		h.logger.Errorf("Failed to get updated record: %v", err)
		c.JSON(http.StatusOK, gin.H{
			"message": "Record updated successfully",
		})
		return
	}

	h.logger.Infof("Updated record %s %s in zone %s", name, recordType, zone)
	c.JSON(http.StatusOK, updatedRecord)
}

// DeleteRecord handles DELETE /api/v1/zones/:zone/records/:name/:type
func (h *Handler) DeleteRecord(c *gin.Context) {
	zone := c.Param("zone")
	name := c.Param("name")
	recordType := knot.RecordType(strings.ToUpper(c.Param("type")))

	if zone == "" || name == "" || recordType == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Zone, name, and type parameters are required",
		})
		return
	}

	if !knot.IsValidRecordType(string(recordType)) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid record type",
		})
		return
	}

	if err := h.knotClient.DeleteRecord(zone, name, recordType); err != nil {
		h.logger.Errorf("Failed to delete record %s %s in zone %s: %v", name, recordType, zone, err)
		if strings.Contains(err.Error(), "zone not allowed") {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Access to zone not allowed",
			})
			return
		}
		if strings.Contains(err.Error(), "record not found") {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Record not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete record",
		})
		return
	}

	h.logger.Infof("Deleted record %s %s from zone %s", name, recordType, zone)
	c.JSON(http.StatusOK, gin.H{
		"message": "Record deleted successfully",
	})
}

// ReloadZone handles POST /api/v1/zones/:zone/reload
func (h *Handler) ReloadZone(c *gin.Context) {
	zone := c.Param("zone")
	if zone == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Zone parameter is required",
		})
		return
	}

	if err := h.knotClient.ReloadZone(zone); err != nil {
		h.logger.Errorf("Failed to reload zone %s: %v", zone, err)
		if strings.Contains(err.Error(), "zone not allowed") {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Access to zone not allowed",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to reload zone",
		})
		return
	}

	h.logger.Infof("Reloaded zone %s", zone)
	c.JSON(http.StatusOK, gin.H{
		"message": "Zone reloaded successfully",
	})
}
