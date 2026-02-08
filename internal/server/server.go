package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/encoding/protojson"

	"grpc-hello/internal/config"
	"grpc-hello/internal/handler"
	"grpc-hello/internal/middleware"
	"grpc-hello/internal/service"

	helloworldpb "grpc-hello/proto/helloworld"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
)

// Server 服务封装
type Server struct {
	cfg             *config.Config
	grpcServer     *grpc.Server
	httpServer     *http.Server
	greetingService *service.GreetingService
	grpcHandler     *handler.GreeterHandler
	httpHandler     *handler.HTTPHandler
	// 用于跟踪服务器启动状态
	started      bool
	startMutex   sync.Mutex
	grpcListener net.Listener
}

// NewServer 创建服务实例
func NewServer(cfg *config.Config) *Server {
	greetingService := service.NewGreetingService(cfg.Features.MaxGreetings)
	grpcHandler := handler.NewGreeterHandler(greetingService)
	httpHandler := handler.NewHTTPHandler(greetingService, cfg)

	return &Server{
		cfg:             cfg,
		greetingService: greetingService,
		grpcHandler:     grpcHandler,
		httpHandler:     httpHandler,
	}
}

// Start 启动服务
func (s *Server) Start() error {
	s.startMutex.Lock()
	defer s.startMutex.Unlock()

	if s.started {
		return fmt.Errorf("server already started")
	}

	// 启动gRPC
	if err := s.startGRPC(); err != nil {
		return fmt.Errorf("failed to start gRPC: %w", err)
	}

	// 启动HTTP
	if err := s.startHTTP(); err != nil {
		// 如果HTTP启动失败，先关闭gRPC
		s.stopGRPC()
		return fmt.Errorf("failed to start HTTP: %w", err)
	}

	s.started = true
	log.Printf("Server started: gRPC=%s, HTTP=%s", s.cfg.GetGRPCAddr(), s.cfg.GetHTTPAddr())

	// 等待退出信号
	s.waitForShutdown()

	return nil
}

// startGRPC 启动gRPC服务
func (s *Server) startGRPC() error {
	// 配置keepalive参数
	keepaliveParams := keepalive.ServerParameters{
		MaxConnectionIdle:     5 * time.Minute,
		MaxConnectionAge:      2 * time.Hour,
		MaxConnectionAgeGrace: 30 * time.Second,
		Time:                  1 * time.Minute,
		Timeout:               20 * time.Second,
	}

	s.grpcServer = grpc.NewServer(
		grpc.MaxRecvMsgSize(10*1024*1024), // 10MB
		grpc.MaxSendMsgSize(10*1024*1024),
		grpc.KeepaliveParams(keepaliveParams),
	)

	helloworldpb.RegisterGreeterServer(s.grpcServer, s.grpcHandler)

	if s.cfg.Features.EnableReflection {
		reflection.Register(s.grpcServer)
	}

	var err error
	s.grpcListener, err = net.Listen("tcp", s.cfg.GetGRPCAddr())
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	go func() {
		log.Printf("gRPC server listening on %s", s.cfg.GetGRPCAddr())
		if err := s.grpcServer.Serve(s.grpcListener); err != nil {
			log.Printf("gRPC server error: %v", err)
		}
	}()

	return nil
}

// startHTTP 启动HTTP服务
func (s *Server) startHTTP() error {
	// 设置Gin模式
	if s.cfg.Server.EnableDebug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建HTTP网关连接
	conn, err := grpc.Dial(
		s.cfg.GetGRPCAddr(),
		grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithTimeout(5*time.Second),
	)
	if err != nil {
		return fmt.Errorf("failed to dial gRPC: %w", err)
	}

	// 确保连接关闭
	defer conn.Close()

	// 创建gRPC网关mux
	gwmux := runtime.NewServeMux(
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
			MarshalOptions: protojson.MarshalOptions{
				UseProtoNames:   true,
				EmitUnpopulated: true,
			},
			UnmarshalOptions: protojson.UnmarshalOptions{
				DiscardUnknown: true,
			},
		}),
	)

	if err := helloworldpb.RegisterGreeterHandler(context.Background(), gwmux, conn); err != nil {
		return fmt.Errorf("failed to register gateway: %w", err)
	}

	// 创建Gin路由
	router := gin.New()
	router.Use(
		middleware.Recovery(),
		middleware.Logger(),
		middleware.RequestID(),
		middleware.CORS(),
		middleware.Timeout(s.cfg.GetTimeout()), // 添加超时中间件
	)

	// 注册路由
	router.Any("/rpc/v1/*any", gin.WrapH(gwmux))
	s.httpHandler.RegisterRoutes(router)

	// HTTP服务器配置优化
	readTimeout := s.cfg.GetTimeout()
	writeTimeout := s.cfg.GetTimeout()

	// HTTP服务器
	s.httpServer = &http.Server{
		Addr:         s.cfg.GetHTTPAddr(),
		Handler:     router,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		// 连接配置
		MaxHeaderBytes: 1 << 20, // 1MB
	}

	go func() {
		log.Printf("HTTP server listening on %s", s.cfg.GetHTTPAddr())
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	return nil
}

// waitForShutdown 等待退出信号并优雅关闭
func (s *Server) waitForShutdown() {
	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, syscall.SIGINT, syscall.SIGTERM)
	<-stopCh

	log.Println("Shutting down servers...")

	// 优雅关闭HTTP
	gracefulTimeout := 10 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), gracefulTimeout)
	defer cancel()

	// 标记服务器正在关闭
	s.startMutex.Lock()
	s.started = false
	s.startMutex.Unlock()

	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(ctx); err != nil {
			log.Printf("HTTP server shutdown error: %v", err)
		} else {
			log.Println("HTTP server stopped gracefully")
		}
	}

	// 优雅关闭gRPC
	if s.grpcServer != nil {
		// 先发送关闭信号
		grpcCh := make(chan struct{})
		go func() {
			s.grpcServer.GracefulStop()
			close(grpcCh)
		}()

		// 等待gRPC关闭或超时
		select {
		case <-grpcCh:
			log.Println("gRPC server stopped gracefully")
		case <-ctx.Done():
			log.Println("gRPC server forced to stop due to timeout")
			s.grpcServer.Stop()
		}
	}

	// 关闭监听器
	if s.grpcListener != nil {
		s.grpcListener.Close()
	}

	log.Println("All servers stopped")
}

// stopGRPC 停止gRPC服务（用于启动失败时清理）
func (s *Server) stopGRPC() {
	if s.grpcServer != nil {
		s.grpcServer.Stop()
		s.grpcServer = nil
	}
	if s.grpcListener != nil {
		s.grpcListener.Close()
		s.grpcListener = nil
	}
}
