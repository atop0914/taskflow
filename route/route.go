package route

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// InitRoutes 路由初始化
func InitRoutes(r *gin.Engine) {
	// Health check - should be at root level
	r.GET("/health", func(c *gin.Context) {
		c.String(200, "OK")
	})

	// Metrics endpoint
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Readiness probe
	r.GET("/ready", func(c *gin.Context) {
		c.String(200, "Ready")
	})

	// Liveness probe
	r.GET("/live", func(c *gin.Context) {
		c.String(200, "Alive")
	})
}
