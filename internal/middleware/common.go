package middleware

import (
	"bytes"
	"io"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequestID 中间件 - 添加请求ID
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Set("trace_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

// Logger 日志中间件
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		method := c.Request.Method

		if query != "" {
			path = path + "?" + query
		}

		log.Printf("[%s] %s %s %d %v | %s",
			c.GetHeader("X-Request-ID"),
			method,
			path,
			status,
			latency,
			c.ClientIP(),
		)
	}
}

// Recovery 恢复中间件（带trace_id）
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				traceID, _ := c.Get("trace_id")
				if traceID == nil {
					traceID = "unknown"
				}
				log.Printf("[%s] Panic recovered: %v", traceID, err)
				c.AbortWithStatusJSON(500, gin.H{
					"code":    500,
					"message": "internal server error",
				})
			}
		}()
		c.Next()
	}
}

// CORS 跨域中间件
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Request-ID")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// RequestBodyLogger 请求体日志中间件（仅开发环境）
func RequestBodyLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == "POST" || c.Request.Method == "PUT" {
			bodyBytes, _ := io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

			if len(bodyBytes) > 0 && len(bodyBytes) < 1024 {
				log.Printf("[%s] Request Body: %s", c.GetHeader("X-Request-ID"), string(bodyBytes))
			}
		}
		c.Next()
	}
}

// Timeout 超时控制中间件
func Timeout(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Simple timeout handling via context
		done := make(chan struct{})

		go func() {
			c.Next()
			close(done)
		}()

		select {
		case <-done:
			// Completed normally
		case <-time.After(timeout):
			log.Printf("[%s] Request timeout after %v", c.GetHeader("X-Request-ID"), timeout)
			c.AbortWithStatusJSON(504, gin.H{
				"code":    504,
				"message": "gateway timeout",
			})
		}
	}
}
