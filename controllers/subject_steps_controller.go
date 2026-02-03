package controllers

import (
	"net/http"

	"Cx_Mcdean_Backend/db"
	"Cx_Mcdean_Backend/models"

	"github.com/gin-gonic/gin"
)

// GET /api/v1/subjects/:subject/steps?active_only=true
func ListSubjectSteps(c *gin.Context) {
	subject := c.Param("subject")
	activeOnly := c.Query("active_only") // 可选：默认不过滤

	q := db.GetDB().Model(&models.DeviceSubjectStep{}).
		Where("subject = ?", subject).
		Order("step_order ASC")

	if activeOnly == "true" {
		q = q.Where("is_active = true")
	}

	var steps []models.DeviceSubjectStep
	if err := q.Find(&steps).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"subject": subject,
		"count":   len(steps),
		"data":    steps,
	})
}
