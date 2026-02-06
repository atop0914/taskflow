package main

import (
	"log"
	"os"

	"grpc-hello/internal/config"
	"grpc-hello/internal/server"
)

func main() {
	// 加载配置
	cfg := config.LoadConfig()

	// 验证配置
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	log.Printf("Starting gRPC-Hello Server...")
	log.Printf("Debug mode: %v", cfg.Server.EnableDebug)
	log.Printf("Features - Stats: %v, Metrics: %v, Reflection: %v",
		cfg.Features.EnableStats,
		cfg.Features.EnableMetrics,
		cfg.Features.EnableReflection,
	)

	// 创建并启动服务器
	srv := server.NewServer(cfg)
	if err := srv.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
		os.Exit(1)
	}
}
