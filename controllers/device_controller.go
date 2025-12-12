package controllers

import (
	"Cx_Mcdean_Backend/config"
	"Cx_Mcdean_Backend/db"
	"Cx_Mcdean_Backend/models"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
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

	if body.Subject == "panel board" {
		_ = setConnectedPolylinesEnergized([]string{body.ID}, body.Energized)
		_ = setConnectedPolylinesEnergizedToday([]string{body.ID}, body.EnergizedToday)
	}
	c.JSON(http.StatusCreated, body)
}

// PUT /api/v1/devices/:id
func UpdateDevice(c *gin.Context) {
	id := c.Param("id")

	// 接收为指针，便于判断“是否传了这个字段”
	type updateDTO struct {
		Text            *string    `json:"text"`
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
	if req.Text != nil {
		changes["text"] = *req.Text
	}
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

	if dev.Subject == "panel board" || dev.Subject == "Breaker" || dev.Subject == "Bus Breaker" || dev.Subject == "transformer" {
		if req.Energized != nil {
			_ = setConnectedPolylinesEnergized([]string{id}, *req.Energized)

		}
		if req.EnergizedToday != nil {
			_ = setConnectedPolylinesEnergizedToday([]string{id}, *req.EnergizedToday)
		}
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
			DoUpdates: clause.AssignmentColumns([]string{"subject", "project", "file_page", "rect_px", "polygon_points_px", "short_segments_px", "text", "comments", "energized", "energized_today", "will_energized_at", "from", "to", "updated_at"}),
		},
	).Create(&arr)

	// 由于 gorm 不直接提供便捷 Upsert 的公用 Clause，另一种更清晰写法是用 gorm.io/gorm/clause
	// 这里保持简洁；如需更严谨，可改为 clause.OnConflict{...}

	if tx.Error != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": tx.Error.Error()})
		return
	}

	var energizedPanels []string
	var deEnergizedPanels []string
	var energizedTodayPanels []string
	var deEnergizedTodayPanels []string
	for _, dev := range arr {
		if dev.Subject == "panel board" {
			if dev.Energized {
				energizedPanels = append(energizedPanels, dev.ID)
			} else {
				deEnergizedPanels = append(deEnergizedPanels, dev.ID)
			}
			if dev.EnergizedToday {
				energizedTodayPanels = append(energizedTodayPanels, dev.ID)
			} else {
				deEnergizedTodayPanels = append(deEnergizedTodayPanels, dev.ID)
			}
		}
	}
	_ = setConnectedPolylinesEnergized(energizedPanels, true)
	_ = setConnectedPolylinesEnergized(deEnergizedPanels, false)

	_ = setConnectedPolylinesEnergizedToday(energizedTodayPanels, true)
	_ = setConnectedPolylinesEnergizedToday(deEnergizedTodayPanels, false)
	c.JSON(http.StatusCreated, gin.H{"count": len(arr)})
}

// 通用函数：把 panel 的某个布尔字段（比如 energized / energized_today）
// 传播到相关 PolyLine 和 Bus 上
func propagatePanelBoolToBusAndPolylines(field string, panelIDs []string, value bool) error {
	if len(panelIDs) == 0 {
		return nil
	}

	dbx := db.GetDB()

	// 因为我们是动态字段名，这里构造一下更新 map
	updateMap := map[string]any{
		field: value,
	}

	// 1️⃣ 更新：所有以 panel 为终点的 PolyLine
	//    subject = 'PolyLine' AND "to" IN panelIDs
	if err := dbx.Model(&models.Device{}).
		Where("subject = ? AND \"to\" IN ?", "PolyLine", panelIDs).
		Updates(updateMap).Error; err != nil {
		return err
	}

	// 2️⃣ 查出：所有 Bus 的 id（以后要用来过滤）
	var allBusIDs []string
	var BUS_SUBJECTS = []string{"Bus"}
	if err := dbx.Model(&models.Device{}).
		Where("subject IN ?", BUS_SUBJECTS).
		Pluck("id", &allBusIDs).Error; err != nil {
		return err
	}
	if len(allBusIDs) == 0 {
		// 没有 Bus，后面就不用干了
		return nil
	}

	// 3️⃣ 查出：和这些 panel 直接相连的 Bus id
	//    条件：subject = 'PolyLine'
	//          "from" IN panelIDs
	//          "to"   IN allBusIDs   （确保真的是连到 Bus 上的线）
	var connectedBusIDs []string
	if err := dbx.Model(&models.Device{}).
		Where("subject = ? AND \"from\" IN ? AND \"to\" IN ?", "PolyLine", panelIDs, allBusIDs).
		Pluck("\"to\"", &connectedBusIDs).Error; err != nil {
		return err
	}
	if len(connectedBusIDs) == 0 {
		// 没有直接相连的 Bus，也不用更新后续
		return nil
	}

	// 4️⃣ 更新：这些 panel → bus 的 PolyLine（同一个字段）
	if err := dbx.Model(&models.Device{}).
		Where("subject = ? AND \"from\" IN ? AND \"to\" IN ?", "PolyLine", panelIDs, connectedBusIDs).
		Updates(updateMap).Error; err != nil {
		return err
	}

	// 5️⃣ 更新：和这些 panel 直接相连的 Bus（同一个字段）
	if err := dbx.Model(&models.Device{}).
		Where("subject = ? AND id IN ?", "Bus", connectedBusIDs).
		Updates(updateMap).Error; err != nil {
		return err
	}

	return nil
}

func setConnectedPolylinesEnergized(panelIDs []string, energized bool) error {
	return propagatePanelBoolToBusAndPolylines("energized", panelIDs, energized)
}

func setConnectedPolylinesEnergizedToday(panelIDs []string, energized_today bool) error {
	return propagatePanelBoolToBusAndPolylines("energized_today", panelIDs, energized_today)
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

// POST /api/v1/devices/:id/files
// Content-Type: multipart/form-data
// 字段：file(文件)，file_type(panel_schedule/test_report/...)
func UploadDeviceFile(c *gin.Context) {
	deviceID := c.Param("id")

	// 1. 先确认 Device 存在
	var dev models.Device
	if err := db.GetDB().First(&dev, "id = ?", deviceID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "device not found"})
		return
	}

	// 2. 取 file_type（可选字段，用于区分 panel schedule / test report）
	fileType := c.PostForm("file_type")
	if fileType == "" {
		fileType = "other"
	}

	// 3. 取文件
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}

	// 4. 构造保存路径
	uploadRoot := config.UploadDir()
	projectDir := filepath.Join(uploadRoot, dev.Project)
	deviceDir := filepath.Join(projectDir, dev.ID)

	if err := os.MkdirAll(deviceDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot create upload dir"})
		return
	}

	// 为了防止重名，在前面加一个时间戳
	timestamp := time.Now().Format("20060102_150405")
	safeName := fmt.Sprintf("%s_%s", timestamp, filepath.Base(fileHeader.Filename))

	dstPath := filepath.Join(deviceDir, safeName)

	// 5. 保存到本地
	if err := c.SaveUploadedFile(fileHeader, dstPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "save file failed"})
		return
	}

	// 6. 写入数据库
	record := models.DeviceFile{
		DeviceID: deviceID,
		Project:  dev.Project,
		FileType: fileType,
		FileName: fileHeader.Filename,
		FilePath: dstPath,
		FileSize: fileHeader.Size,
		MimeType: fileHeader.Header.Get("Content-Type"),
	}

	if err := db.GetDB().Create(&record).Error; err != nil {
		// 数据库失败的话，把刚刚保存的文件删掉，避免垃圾文件
		_ = os.Remove(dstPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db save failed"})
		return
	}

	c.JSON(http.StatusCreated, record)
}

// GET /api/v1/devices/:id/files
func ListDeviceFiles(c *gin.Context) {
	deviceID := c.Param("id")

	var files []models.DeviceFile
	if err := db.GetDB().
		Where("device_id = ?", deviceID).
		Order("created_at DESC").
		Find(&files).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"device_id": deviceID,
		"count":     len(files),
		"data":      files,
	})
}

// GET /api/v1/files/:id
func DownloadDeviceFile(c *gin.Context) {
	id := c.Param("id")

	var f models.DeviceFile
	if err := db.GetDB().First(&f, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", f.FileName))
	c.Header("Content-Type", f.MimeType)
	c.File(f.FilePath)
}

// DELETE /api/v1/files/:id
func DeleteDeviceFile(c *gin.Context) {
	id := c.Param("id")

	var f models.DeviceFile
	if err := db.GetDB().First(&f, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}

	if err := db.GetDB().Delete(&f).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db delete failed"})
		return
	}

	_ = os.Remove(f.FilePath) // 文件删不掉也不影响接口返回

	c.Status(http.StatusNoContent)
}
