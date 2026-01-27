package controllers

import (
	"Cx_Mcdean_Backend/db"
	"Cx_Mcdean_Backend/models"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type deviceTemplatePayload struct {
	Name    string   `json:"name"`
	Subject string   `json:"subject"`
	Steps   []string `json:"steps"`
}

// GET /api/v1/device-templates
// optional: ?subject=ATS
func ListDeviceTemplates(c *gin.Context) {
	subject := c.Query("subject")
	var templates []models.DeviceTemplate
	query := db.GetDB().Order("updated_at DESC")
	if subject != "" {
		query = query.Where("subject = ?", subject)
	}
	if err := query.Find(&templates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": templates})
}

// GET /api/v1/device-templates/:id
func GetDeviceTemplate(c *gin.Context) {
	id := c.Param("id")
	var template models.DeviceTemplate
	if err := db.GetDB().First(&template, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, template)
}

// POST /api/v1/device-templates
func CreateDeviceTemplate(c *gin.Context) {
	var payload deviceTemplatePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if payload.Name == "" || payload.Subject == "" || len(payload.Steps) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name, subject, and steps are required"})
		return
	}
	stepsJSON, err := json.Marshal(payload.Steps)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid steps"})
		return
	}

	template := models.DeviceTemplate{
		Name:    payload.Name,
		Subject: payload.Subject,
		Steps:   datatypes.JSON(stepsJSON),
	}

	if err := db.GetDB().Create(&template).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, template)
}

// PUT /api/v1/device-templates/:id
func UpdateDeviceTemplate(c *gin.Context) {
	id := c.Param("id")
	var payload deviceTemplatePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var template models.DeviceTemplate
	if err := db.GetDB().First(&template, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	changes := map[string]any{}
	if payload.Name != "" {
		changes["name"] = payload.Name
	}
	if payload.Subject != "" {
		changes["subject"] = payload.Subject
	}
	if payload.Steps != nil {
		if len(payload.Steps) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "steps cannot be empty"})
			return
		}
		stepsJSON, err := json.Marshal(payload.Steps)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid steps"})
			return
		}
		changes["steps"] = datatypes.JSON(stepsJSON)
	}

	if len(changes) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no fields to update"})
		return
	}

	if err := db.GetDB().Model(&template).Updates(changes).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := db.GetDB().First(&template, "id = ?", id).Error; err == nil {
		c.JSON(http.StatusOK, template)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// DELETE /api/v1/device-templates/:id
func DeleteDeviceTemplate(c *gin.Context) {
	id := c.Param("id")
	if err := db.GetDB().Delete(&models.DeviceTemplate{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
