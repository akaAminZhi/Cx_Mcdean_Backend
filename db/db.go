package db

import (
	"log"
	"time"

	"github.com/lib/pq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"Cx_Mcdean_Backend/config"
	"Cx_Mcdean_Backend/models"
)

var instance *gorm.DB

func Connect() (*gorm.DB, error) {
	if instance != nil {
		return instance, nil
	}

	dsn := config.GetDSN()
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, err
	}

	// 连接池设置
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetConnMaxLifetime(60 * time.Minute)

	// 自动迁移
	if err := db.AutoMigrate(&models.Device{}); err != nil {
		return nil, err
	}
	// 在 Connect() 成功后，AutoMigrate 之后，追加：
	if err := db.Exec(`CREATE EXTENSION IF NOT EXISTS pg_trgm`).Error; err != nil {
		return nil, err
	}
	// 针对 text 字段创建 GIN 三元组索引（适合 ILIKE、相似搜索）
	if err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_devices_text_trgm ON devices USING gin (text gin_trgm_ops)`).Error; err != nil {
		return nil, err
	}
	// 如果你经常按 project + text 搜，可以建联合索引（可选）
	if err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_devices_project_text_trgm ON devices USING gin (project gin_trgm_ops, text gin_trgm_ops)`).Error; err != nil {
		return nil, err
	}

	instance = db
	return instance, nil
}

func GetDB() *gorm.DB {
	if instance == nil {
		log.Fatal("DB not initialized. Call db.Connect() first.")
	}
	return instance
}

// 仅用于静态引用，避免编译器去掉 pq 的数组类型映射
var _ = pq.Int64Array{}
