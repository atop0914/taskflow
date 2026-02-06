package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
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
	grpcServer      *grpc.Server
	httpServer      *http.Server
	greetingService *service.GreetingService
	grpcHandler     *handler.GreeterHandler
	httpHandler     *handler.HTTPHandler
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
	// 启动gRPC
	if err := s.startGRPC(); err != nil {
		return fmt.Errorf("failed to start gRPC: %w", err)
	}

	// 启动HTTP
	if err := s.startHTTP(); err != nil {
		return fmt.Errorf("failed to start HTTP: %w", err)
	}

	log.Printf("Server started: gRPC=%s, HTTP=%s", s.cfg.GetGRPCAddr(), s.cfg.GetHTTPAddr())

	// 等待退出信号
	s.waitForShutdown()

	return nil
}

// startGRPC 启动gRPC服务
func (s *Server) startGRPC() error {
	s.grpcServer = grpc.NewServer(
		grpc.MaxRecvMsgSize(10*1024*1024), // 10MB
		grpc.MaxSendMsgSize(10*1024*1024),
	)

	helloworldpb.RegisterGreeterServer(s.grpcServer, s.grpcHandler)

	if s.cfg.Features.EnableReflection {
		reflection.Register(s.grpcServer)
	}

	lis, err := net.Listen("tcp", s.cfg.GetGRPCAddr())
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	go func() {
		log.Printf("gRPC server listening on %s", s.cfg.GetGRPCAddr())
		if err := s.grpcServer.Serve(lis); err != nil {
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
	)

	// 注册路由
	router.Any("/rpc/v1/*any", gin.WrapH(gwmux))
	s.httpHandler.RegisterRoutes(router)

	// HTTP服务器
	s.httpServer = &http.Server{
		Addr:         s.cfg.GetHTTPAddr(),
		Handler:     router,
		ReadTimeout: s.cfg.GetTimeout(),
		WriteTimeout: s.cfg.GetTimeout(),
	}

	go func() {
		log.Printf("HTTP server listening on %s", s.cfg.GetHTTPAddr())
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	return nil
}

// waitForShutdown 等待退出信号
func (s *Server) waitForShutdown() {
	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, syscall.SIGINT, syscall.SIGTERM)
	<-stopCh

	log.Println("Shutting down servers...")

	// 优雅关闭HTTP
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	// 优雅关闭gRPC
	s.grpcServer.GracefulStop()

	log.Println("Servers stopped gracefully")
}
