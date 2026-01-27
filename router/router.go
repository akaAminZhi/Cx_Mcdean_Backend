package router

import (
	"Cx_Mcdean_Backend/config"
	"Cx_Mcdean_Backend/controllers"
	"Cx_Mcdean_Backend/middleware"

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

	// 初始化 Entra JWT Middleware

	_, _, err := middleware.NewEntraJWTMiddleware(middleware.EntraJWTConfig{
		TenantID: config.C.AzureTenantID,
		Issuer:   config.C.AzureIssuer,
		Audience: config.C.AzureAudience,
		JWKSURL:  config.AzureJWKSURL(),
	})
	if err != nil {
		panic(err) // 启动就失败，避免“没鉴权就上线”
	}

	v1 := r.Group("/api/v1")
	// v1.Use(jwtMW)
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

			dev.POST("/:id/files", controllers.UploadDeviceFile)
			dev.GET("/:id/files", controllers.ListDeviceFiles)
		}

		// ✅ 文件：按 fileId 下载 / 删除
		v1.GET("/files/:id", controllers.DownloadDeviceFile)
		v1.DELETE("/files/:id", controllers.DeleteDeviceFile)
		// 新增：按项目名查找all设备
		v1.GET("/projects/:project/devices", controllers.GetDevicesByProject)
		// 新增：按项目名查找 specific equipments
		v1.GET("/projects/:project/equipments", controllers.GetEquipmentsByProject)

	}

	return r
}
