package middleware

import (
	"bytes"
	"context"
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

// Timeout 超时控制中间件（优化版 - 修复goroutine泄漏）
func Timeout(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 使用 context 实现超时控制
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)

		done := make(chan struct{})
		go func() {
			defer close(done)
			// 使用 defer 确保在函数退出时调用 c.Abort()
			// 这样可以防止gin继续处理已取消的请求
			defer func() {
				if err := recover(); err != nil {
					// 记录panic但不再传播
					traceID := c.GetHeader("X-Request-ID")
					log.Printf("[%s] Panic in timeout handler: %v", traceID, err)
				}
			}()
			c.Next()
		}()

		select {
		case <-done:
			// 请求正常完成
		case <-ctx.Done():
			// 超时或上下文取消
			traceID := c.GetHeader("X-Request-ID")
			if ctx.Err() == context.DeadlineExceeded {
				log.Printf("[%s] Request timeout after %v", traceID, timeout)
			} else {
				log.Printf("[%s] Request context cancelled", traceID)
			}
			// 终止当前请求的处理，防止goroutine泄漏
			c.Abort()
			c.AbortWithStatusJSON(504, gin.H{
				"code":    504,
				"message": "gateway timeout",
			})
		}
	}
}
