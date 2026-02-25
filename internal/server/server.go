package server

import (
	"context"
	"fmt"
	"net/http"
	"net"
	"os"
	"os/signal"
	path2 "path"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"

	"taskflow/internal/config"
	"taskflow/internal/handler"
	"taskflow/internal/logger"
	"taskflow/internal/middleware"
	"taskflow/internal/model"
	"taskflow/internal/repository"
	pb "taskflow/proto"
)

// Server HTTP/gRPC服务封装
type Server struct {
	cfg        *config.Config
	httpServer *http.Server
	grpcServer *grpc.Server
	started    bool
	startMutex sync.Mutex
	taskHandler *handler.TaskHandler
	taskRepo    *repository.TaskRepository
}

// NewServer 创建服务实例
func NewServer(cfg *config.Config) *Server {
	return &Server{
		cfg: cfg,
	}
}

// Start 启动服务
func (s *Server) Start() error {
	s.startMutex.Lock()
	defer s.startMutex.Unlock()

	if s.started {
		return fmt.Errorf("server already started")
	}

	// 获取数据库路径（支持环境变量 TASKFLOW_DB_PATH）
	dbPath := s.cfg.Server.DBPath
	// 处理用户主目录
	if strings.HasPrefix(dbPath, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			homeDir = "."
		}
		dbPath = path2.Join(homeDir, strings.TrimPrefix(dbPath, "~/"))
	}

	// 确保目录存在
	dbDir := path2.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return fmt.Errorf("failed to create db directory: %w", err)
	}

	db, err := repository.NewSQLite(dbPath)
	if err != nil {
		return fmt.Errorf("failed to init database: %w", err)
	}
	defer db.Close()

	// 初始化表结构
	if err := db.InitSchema(); err != nil {
		return fmt.Errorf("failed to init schema: %w", err)
	}

	taskRepo := repository.NewTaskRepository(db)
	s.taskRepo = taskRepo
	s.taskHandler = handler.NewTaskHandler(taskRepo)

	// 启动 gRPC 服务器
	if err := s.startGRPC(); err != nil {
		return fmt.Errorf("failed to start gRPC: %w", err)
	}

	// 启动 HTTP 服务器
	if err := s.startHTTP(); err != nil {
		return fmt.Errorf("failed to start HTTP: %w", err)
	}

	s.started = true
	logger.Infof("Server started: gRPC=%s, HTTP=%s", s.cfg.GetGRPCAddr(), s.cfg.GetHTTPAddr())

	s.waitForShutdown()

	return nil
}

// startGRPC 启动gRPC服务
func (s *Server) startGRPC() error {
	lis, err := net.Listen("tcp", s.cfg.GetGRPCAddr())
	if err != nil {
		return fmt.Errorf("failed to listen on gRPC: %w", err)
	}

	// 创建 gRPC 服务器
	s.grpcServer = grpc.NewServer()
	
	// 注册 TaskService
	pb.RegisterTaskServiceServer(s.grpcServer, s.taskHandler)

	go func() {
		logger.Infof("gRPC server listening on %s", s.cfg.GetGRPCAddr())
		if err := s.grpcServer.Serve(lis); err != nil {
			logger.Errorf("gRPC server error: %v", err)
		}
	}()

	return nil
}

// startHTTP 启动HTTP服务
func (s *Server) startHTTP() error {
	if s.cfg.Server.EnableDebug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.RemoveExtraSlash = true
	router.Use(
		middleware.Recovery(),
		middleware.Logger(),
		middleware.RequestID(),
		middleware.CORS(),
		middleware.Timeout(s.cfg.GetTimeout()),
	)

	// 健康检查
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Prometheus 指标端点
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// 注册 API 路由
	if s.taskHandler != nil {
		s.registerRoutes(router)
	}

	s.httpServer = &http.Server{
		Addr:           s.cfg.GetHTTPAddr(),
		Handler:        router,
		ReadTimeout:    s.cfg.GetTimeout(),
		WriteTimeout:   s.cfg.GetTimeout(),
		MaxHeaderBytes: 1 << 20,
	}

	go func() {
		logger.Infof("HTTP server listening on %s", s.cfg.GetHTTPAddr())
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Errorf("HTTP server error: %v", err)
		}
	}()

	return nil
}

// registerRoutes 注册路由
func (s *Server) registerRoutes(router *gin.Engine) {
	// 任务列表
	router.GET("/api/v1/tasks", s.handleListTasks)
	router.POST("/api/v1/tasks", s.handleCreateTask)
	
	// 单个任务操作
	router.GET("/api/v1/tasks/:id", s.handleGetTask)
	router.PUT("/api/v1/tasks/:id", s.handleUpdateTask)
	
	// 任务统计
	router.GET("/api/v1/tasks/stats", s.handleTaskStats)
}

// handleCreateTask 创建任务
func (s *Server) handleCreateTask(c *gin.Context) {
	var req struct {
		Name         string            `json:"name" binding:"required"`
		Description  string            `json:"description"`
		Priority     int32             `json:"priority"`
		TaskType     string            `json:"task_type"`
		InputParams  map[string]string `json:"input_params"`
		Dependencies []string          `json:"dependencies"`
		MaxRetries   int32             `json:"max_retries"`
		CreatedBy    string            `json:"created_by"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 1001, "message": "invalid request: " + err.Error()})
		return
	}

	pbReq := &pb.CreateTaskRequest{
		Name:         req.Name,
		Description:  req.Description,
		Priority:     pb.TaskPriority(req.Priority),
		TaskType:     req.TaskType,
		InputParams:  req.InputParams,
		Dependencies: req.Dependencies,
		MaxRetries:   req.MaxRetries,
		CreatedBy:    req.CreatedBy,
	}

	task, err := s.taskHandler.CreateTask(c.Request.Context(), pbReq)
	if err != nil {
		c.JSON(500, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(201, task)
}

// handleListTasks 列出任务
func (s *Server) handleListTasks(c *gin.Context) {
	page := int32(parseInt(c.Query("page"), 1))
	pageSize := int32(parseInt(c.Query("page_size"), 20))
	keyword := c.Query("keyword")
	taskType := c.Query("type")
	statusVal := c.Query("status")
	priorityStr := c.Query("priority")

	req := &pb.ListTasksRequest{
		Page:     page,
		PageSize: pageSize,
		Keyword:  keyword,
		TaskType: taskType,
	}

	if statusVal != "" {
		if v := parseInt(statusVal, -1); v > 0 {
			req.StatusFilter = []pb.TaskStatus{pb.TaskStatus(v)}
		}
	}
	if priorityStr != "" {
		req.Priority = pb.TaskPriority(parseInt(priorityStr, 0))
	}

	resp, err := s.taskHandler.ListTasks(c.Request.Context(), req)
	if err != nil {
		c.JSON(500, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(200, resp)
}

// handleGetTask 获取任务
func (s *Server) handleGetTask(c *gin.Context) {
	id := c.Param("id")
	includeEvents := c.Query("include_events") == "true"

	req := &pb.GetTaskRequest{
		Id:            id,
		IncludeEvents: includeEvents,
	}

	task, err := s.taskHandler.GetTask(c.Request.Context(), req)
	if err != nil {
		c.JSON(500, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(200, task)
}

// handleUpdateTask 更新任务
func (s *Server) handleUpdateTask(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Status       int32             `json:"status"`
		OutputResult map[string]string `json:"output_result"`
		ErrorMessage string            `json:"error_message"`
		RetryCount   int32             `json:"retry_count"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 1001, "message": "invalid request: " + err.Error()})
		return
	}

	pbReq := &pb.UpdateTaskRequest{
		Id:           id,
		Status:       pb.TaskStatus(req.Status),
		OutputResult: req.OutputResult,
		ErrorMessage: req.ErrorMessage,
		RetryCount:   req.RetryCount,
	}

	task, err := s.taskHandler.UpdateTask(c.Request.Context(), pbReq)
	if err != nil {
		c.JSON(500, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(200, task)
}

// handleTaskStats 任务统计
func (s *Server) handleTaskStats(c *gin.Context) {
	// 获取各状态的任务数量
	pending := model.TaskStatusPending
	running := model.TaskStatusRunning
	succeeded := model.TaskStatusSucceeded
	failed := model.TaskStatusFailed
	cancelled := model.TaskStatusCancelled

	pendingCount, _ := s.taskRepo.Count(&pending)
	runningCount, _ := s.taskRepo.Count(&running)
	succeededCount, _ := s.taskRepo.Count(&succeeded)
	failedCount, _ := s.taskRepo.Count(&failed)
	cancelledCount, _ := s.taskRepo.Count(&cancelled)

	total := pendingCount + runningCount + succeededCount + failedCount + cancelledCount

	c.JSON(200, gin.H{
		"total":      total,
		"pending":    pendingCount,
		"running":    runningCount,
		"succeeded":  succeededCount,
		"failed":     failedCount,
		"cancelled":  cancelledCount,
	})
}

// waitForShutdown 等待退出信号并优雅关闭
func (s *Server) waitForShutdown() {
	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, syscall.SIGINT, syscall.SIGTERM)
	<-stopCh

	logger.Info("Shutting down server...")

	gracefulTimeout := 10 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), gracefulTimeout)
	defer cancel()

	s.startMutex.Lock()
	s.started = false
	s.startMutex.Unlock()

	// 优雅关闭 gRPC
	if s.grpcServer != nil {
		s.grpcServer.GracefulStop()
		logger.Info("gRPC server stopped gracefully")
	}

	// 优雅关闭 HTTP
	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(ctx); err != nil {
			logger.Errorf("HTTP server shutdown error: %v", err)
		} else {
			logger.Info("HTTP server stopped gracefully")
		}
	}

	// 同步日志
	logger.Sync()
	logger.Info("Server stopped")
}

// GetHTTPAddr 获取HTTP地址
func (s *Server) GetHTTPAddr() string {
	return s.cfg.GetHTTPAddr()
}

// parseInt 解析整数
func parseInt(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	var n int
	fmt.Sscanf(s, "%d", &n)
	if n == 0 {
		return defaultVal
	}
	return n
}
