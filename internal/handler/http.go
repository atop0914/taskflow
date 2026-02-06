package handler

import (
	"net/http"
	"strconv"
	"time"

	"grpc-hello/api/dto"
	"grpc-hello/internal/config"
	"grpc-hello/internal/service"

	"github.com/gin-gonic/gin"
)

// HTTPHandler HTTP路由处理器
type HTTPHandler struct {
	greetingService *service.GreetingService
	cfg             *config.Config
	startTime       time.Time
}

// NewHTTPHandler 创建HTTP处理器
func NewHTTPHandler(greetingService *service.GreetingService, cfg *config.Config) *HTTPHandler {
	return &HTTPHandler{
		greetingService: greetingService,
		cfg:             cfg,
		startTime:       time.Now(),
	}
}

// RegisterRoutes 注册路由
func (h *HTTPHandler) RegisterRoutes(r *gin.Engine) {
	// 健康检查
	r.GET("/health", h.Health)

	// API v1
	v1 := r.Group("/rpc/v1")
	{
		v1.POST("/sayHello", h.SayHello)
		v1.POST("/sayHelloMultiple", h.SayHelloMultiple)
		v1.GET("/greetingStats", h.GetGreetingStats)
	}
}

// Health 健康检查
func (h *HTTPHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, dto.NewSuccessResponse(&dto.HealthResponse{
		Status:    "ok",
		GRPCPort:  h.cfg.Server.GRPCPort,
		HTTPPort:  h.cfg.Server.HTTPPort,
		Uptime:    int64(time.Since(h.startTime).Seconds()),
		Timestamp: time.Now().Unix(),
	}))
}

// SayHello 问候接口
func (h *HTTPHandler) SayHello(c *gin.Context) {
	var req dto.HelloRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse(
			int(dto.CodeBadRequest),
			"invalid request body",
		))
		return
	}

	name := req.Name
	if name == "" {
		name = "World"
	}

	h.greetingService.UpdateStats(name)
	message := h.greetingService.BuildMessage(name, req.Language, "")

	c.JSON(http.StatusOK, dto.NewSuccessResponse(&dto.HelloResponse{
		Message:   message,
		Timestamp: time.Now().Unix(),
		Language:  req.Language,
	}))
}

// SayHelloMultiple 批量问候接口
func (h *HTTPHandler) SayHelloMultiple(c *gin.Context) {
	var req dto.HelloMultipleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse(
			int(dto.CodeBadRequest),
			"invalid request body",
		))
		return
	}

	maxGreetings := h.greetingService.GetMaxGreetings()
	if len(req.Names) > maxGreetings {
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse(
			int(dto.CodeTooManyNames),
			dto.ErrTooManyNames.Message+" (max: "+strconv.Itoa(maxGreetings)+")",
		))
		return
	}

	var greetings []*dto.HelloResponse
	for _, name := range req.Names {
		h.greetingService.UpdateStats(name)
		message := h.greetingService.BuildMessage(name, req.Language, req.CommonMessage)

		greetings = append(greetings, &dto.HelloResponse{
			Message:   message,
			Timestamp: time.Now().Unix(),
			Language:  req.Language,
		})
	}

	c.JSON(http.StatusOK, dto.NewSuccessResponse(&dto.HelloMultipleResponse{
		Greetings:  greetings,
		TotalCount: int32(len(greetings)),
	}))
}

// GetGreetingStats 获取统计信息
func (h *HTTPHandler) GetGreetingStats(c *gin.Context) {
	filter := c.Query("filter")

	totalReq, uniqueNames, nameFreq, lastReq := h.greetingService.GetStats(filter, 10)

	c.JSON(http.StatusOK, dto.NewSuccessResponse(&dto.StatsResponse{
		TotalRequests:   totalReq,
		UniqueNames:     uniqueNames,
		NameFrequency:   nameFreq,
		LastRequestTime: lastReq,
	}))
}
