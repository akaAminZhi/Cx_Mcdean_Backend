package main

import (
	"log"
	"time"

	"Cx_Mcdean_Backend/config"
	"Cx_Mcdean_Backend/db"
	"Cx_Mcdean_Backend/models"
)

// FirstStepRow represents first step per subject
type FirstStepRow struct {
	Subject  string `gorm:"column:subject"`
	FirstKey string `gorm:"column:first_key"`
}

func main() {
	config.Load()

	dbx, err := db.Connect()
	if err != nil {
		log.Fatalf("connect db failed: %v", err)
	}

	// 1️⃣ 查询每个 subject 的第一步（is_active=true 且 step_order 最小）
	var rows []FirstStepRow
	err = dbx.Raw(`
		SELECT DISTINCT ON (subject)
		       subject,
		       key AS first_key
		FROM device_subject_steps
		WHERE is_active = true
		ORDER BY subject, step_order ASC
	`).Scan(&rows).Error
	if err != nil {
		log.Fatalf("query first steps failed: %v", err)
	}
	if len(rows) == 0 {
		log.Fatal("no active steps found in device_subject_steps")
	}

	// 2️⃣ 建立 subject -> firstStepKey 映射
	firstStepMap := make(map[string]string, len(rows))
	for _, r := range rows {
		firstStepMap[r.Subject] = r.FirstKey
	}

	// 3️⃣ 回填 devices.current_status（仅为空 / NULL 的）
	var totalUpdated int64
	for subject, firstKey := range firstStepMap {
		if firstKey == "" {
			continue
		}

		res := dbx.Model(&models.Device{}).
			Where("subject = ? AND (current_status IS NULL OR current_status = '')", subject).
			Updates(map[string]any{
				"current_status": firstKey,
				"updated_at":     time.Now(),
			})
		if res.Error != nil {
			log.Fatalf("update failed for subject=%q: %v", subject, res.Error)
		}

		log.Printf(
			"subject=%q first_step=%q updated=%d",
			subject,
			firstKey,
			res.RowsAffected,
		)

		totalUpdated += res.RowsAffected
	}

	log.Printf("DONE ✅ total devices updated=%d", totalUpdated)
}
