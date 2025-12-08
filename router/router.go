package router

import (
	"Cx_Mcdean_Backend/controllers"
	"time"

	"github.com/gin-gonic/gin"
)

func Setup() *gin.Engine {
	r := gin.Default()

	// 允许前端跨域（简单示例）
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// 简单的请求计时中间件（可选）
	r.Use(func(c *gin.Context) {
		start := time.Now()
		c.Next()
		_ = time.Since(start)
	})

	r.GET("/healthz", controllers.Health)

	v1 := r.Group("/api/v1")
	{
		dev := v1.Group("/devices")
		{
			dev.GET("", controllers.ListDevices)
			dev.GET("/:id", controllers.GetDevice)
			dev.POST("", controllers.CreateDevice)
			dev.PUT("/:id", controllers.UpdateDevice)
			dev.DELETE("/:id", controllers.DeleteDevice)
			dev.POST("/import", controllers.ImportDevices)

			// 新增：模糊搜索
			dev.GET("/search", controllers.SearchDevices)
		}
		// 新增：按项目名查找设备
		v1.GET("/projects/:project/devices", controllers.GetDevicesByProject)
	}

	return r
}
