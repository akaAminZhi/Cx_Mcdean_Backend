package controllers

import (
	"Cx_Mcdean_Backend/db"
	"Cx_Mcdean_Backend/models"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PaginationQuery struct {
	Page int `form:"page,default=1"`
	Size int `form:"size,default=20"`
}

func Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// GET /api/v1/devices
func ListDevices(c *gin.Context) {
	var q PaginationQuery
	if err := c.ShouldBindQuery(&q); err != nil || q.Page < 1 || q.Size < 1 || q.Size > 1000 {
		q = PaginationQuery{Page: 1, Size: 20}
	}
	var items []models.Device
	var total int64

	d := db.GetDB()
	d.Model(&models.Device{}).Count(&total)

	offset := (q.Page - 1) * q.Size
	if err := d.Order("updated_at DESC").Limit(q.Size).Offset(offset).Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data":       items,
		"pagination": gin.H{"page": q.Page, "size": q.Size, "total": total},
	})
}

// GET /api/v1/devices/:id
func GetDevice(c *gin.Context) {
	id := c.Param("id")
	var dev models.Device
	if err := db.GetDB().First(&dev, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, dev)
}

// GET /api/v1/projects/:project/devices
func GetDevicesByProject(c *gin.Context) {
	project := c.Param("project")
	var devices []models.Device

	if err := db.GetDB().Where("project = ?", project).Order("updated_at DESC").Find(&devices).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"project": project,
		"count":   len(devices),
		"data":    devices,
	})
}

// POST /api/v1/devices
func CreateDevice(c *gin.Context) {
	var body models.Device
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if body.ID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return
	}
	if err := db.GetDB().Create(&body).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, body)
}

// PUT /api/v1/devices/:id
func UpdateDevice(c *gin.Context) {
	id := c.Param("id")

	// 接收为指针，便于判断“是否传了这个字段”
	type updateDTO struct {
		Comments        *string    `json:"comments"`
		Energized       *bool      `json:"energized"`
		EnergizedToday  *bool      `json:"energized_today"`
		WillEnergizedAt *time.Time `json:"will_energized_at"`
	}
	var req updateDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var dev models.Device
	if err := db.GetDB().First(&dev, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	changes := map[string]any{}
	if req.Comments != nil {
		changes["comments"] = *req.Comments
	}
	if req.Energized != nil {
		changes["energized"] = *req.Energized // false 也会被更新
	}
	if req.EnergizedToday != nil {
		changes["energized_today"] = *req.EnergizedToday
	}
	if req.WillEnergizedAt != nil {
		changes["will_energized_at"] = *req.WillEnergizedAt
	}

	if len(changes) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no fields to update"})
		return
	}

	if err := db.GetDB().Model(&dev).Updates(changes).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := db.GetDB().First(&dev, "id = ?", id).Error; err == nil {
		c.JSON(http.StatusOK, dev)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// DELETE /api/v1/devices/:id
func DeleteDevice(c *gin.Context) {
	id := c.Param("id")
	if err := db.GetDB().Delete(&models.Device{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// POST /api/v1/devices/import
// 请求体为 JSON 数组（即你给的那段）
func ImportDevices(c *gin.Context) {
	var arr []models.Device
	if err := c.ShouldBindJSON(&arr); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(arr) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "empty array"})
		return
	}
	tx := db.GetDB().Clauses(
		clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{"subject", "project", "file_page", "rect_px", "polygon_points_px", "text", "comments", "energized", "energized_today", "will_energized_at", "from", "to", "updated_at"}),
		},
	).Create(&arr)

	// 由于 gorm 不直接提供便捷 Upsert 的公用 Clause，另一种更清晰写法是用 gorm.io/gorm/clause
	// 这里保持简洁；如需更严谨，可改为 clause.OnConflict{...}

	if tx.Error != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": tx.Error.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"count": len(arr)})
}

// GET /api/v1/devices/search?q=LAB25E&page=1&size=20&project=LAB25&file_page=1
func SearchDevices(c *gin.Context) {
	type Query struct {
		Q        string `form:"q" binding:"required"`
		Project  string `form:"project"`
		FilePage int    `form:"file_page"`
		Page     int    `form:"page,default=1"`
		Size     int    `form:"size,default=20"`
	}

	var q Query
	if err := c.ShouldBindQuery(&q); err != nil || q.Q == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "q is required"})
		return
	}
	if q.Page < 1 {
		q.Page = 1
	}
	if q.Size < 1 || q.Size > 200 {
		q.Size = 20
	}

	d := db.GetDB().Model(&models.Device{})

	// 必填：对 text 做 ILIKE 模糊匹配
	d = d.Where("text ILIKE ?", "%"+q.Q+"%")

	// 可选：附加过滤
	if q.Project != "" {
		d = d.Where("project = ?", q.Project)
	}
	if q.FilePage != 0 {
		d = d.Where("file_page = ?", q.FilePage)
	}

	// 统计总数
	var total int64
	if err := d.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 查询数据
	var items []models.Device
	offset := (q.Page - 1) * q.Size
	if err := d.Order("updated_at DESC").Limit(q.Size).Offset(offset).Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"query": gin.H{
			"q":         q.Q,
			"project":   q.Project,
			"file_page": q.FilePage,
			"page":      q.Page,
			"size":      q.Size,
		},
		"pagination": gin.H{"total": total},
		"data":       items,
	})
}
