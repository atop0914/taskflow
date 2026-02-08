package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// 端口范围常量
const (
	MinPort = 1
	MaxPort = 65535
)

// 默认配置常量
const (
	// Server defaults
	DefaultGRPCPort     = "8080"
	DefaultHTTPPort     = "8090"
	DefaultTimeout      = 30  // seconds
	DefaultMaxConns     = 1000
	DefaultLogLevel     = "info"
	DefaultMaxGreetings = 100
)

// ServerConfig 服务配置
//goland:noinspection GoDeprecation
type ServerConfig struct {
	GRPCPort    string `yaml:"grpc_port" env:"GRPC_PORT"`       // gRPC服务端口 (1-65535)
	HTTPPort    string `yaml:"http_port" env:"HTTP_PORT"`       // HTTP服务端口 (1-65535)
	EnableDebug bool   `yaml:"enable_debug" env:"ENABLE_DEBUG"` // 启用调试模式
	Timeout     int    `yaml:"timeout" env:"SERVER_TIMEOUT"`   // 请求超时时间（秒），默认30秒
	MaxConns    int    `yaml:"max_conns" env:"MAX_CONNECTIONS"` // 最大连接数，默认1000
	LogLevel    string `yaml:"log_level" env:"LOG_LEVEL"`      // 日志级别：debug, info, warn, error
}

// FeatureFlags 功能开关
type FeatureFlags struct {
	EnableReflection bool `yaml:"enable_reflection" env:"ENABLE_REFLECTION"` // 启用gRPC反射
	EnableStats      bool `yaml:"enable_stats" env:"ENABLE_STATS"`          // 启用统计功能
	EnableMetrics    bool `yaml:"enable_metrics" env:"METRICS_ENABLED"`     // 启用Prometheus指标
	MaxGreetings     int  `yaml:"max_greetings" env:"MAX_GREETINGS"`        // 最大问候数量，默认100
}

// Config 配置
type Config struct {
	Server   ServerConfig  `yaml:"server"`
	Features FeatureFlags  `yaml:"features"`
	mu       sync.RWMutex  // 用于配置热加载
}

// LoadConfig 加载配置（支持环境变量覆盖）
// 环境变量优先级高于配置文件默认值
func LoadConfig() *Config {
	cfg := &Config{
		Server: ServerConfig{
			GRPCPort:    getEnv("GRPC_PORT", DefaultGRPCPort),
			HTTPPort:    getEnv("HTTP_PORT", DefaultHTTPPort),
			EnableDebug: getEnvBool("ENABLE_DEBUG"),
			Timeout:     getEnvInt("SERVER_TIMEOUT", DefaultTimeout),
			MaxConns:    getEnvInt("MAX_CONNECTIONS", DefaultMaxConns),
			LogLevel:    getEnv("LOG_LEVEL", DefaultLogLevel),
		},
		Features: FeatureFlags{
			EnableReflection: getEnvBool("ENABLE_REFLECTION"),
			EnableStats:      getEnvBool("ENABLE_STATS"),
			EnableMetrics:    getEnvBool("METRICS_ENABLED"),
			MaxGreetings:     getEnvInt("MAX_GREETINGS", DefaultMaxGreetings),
		},
	}
	return cfg
}

// Validate 验证配置（包含端口范围验证）
func (c *Config) Validate() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var errs []string

	// 验证gRPC端口范围
	if err := validatePort(c.Server.GRPCPort, "GRPC_PORT"); err != nil {
		errs = append(errs, err.Error())
	}

	// 验证HTTP端口范围
	if err := validatePort(c.Server.HTTPPort, "HTTP_PORT"); err != nil {
		errs = append(errs, err.Error())
	}

	// 验证MaxGreetings
	if c.Features.MaxGreetings <= 0 {
		errs = append(errs, fmt.Sprintf("MAX_GREETINGS must be greater than 0, got %d", c.Features.MaxGreetings))
	}

	// 验证Timeout
	if c.Server.Timeout <= 0 {
		errs = append(errs, fmt.Sprintf("SERVER_TIMEOUT must be greater than 0, got %d", c.Server.Timeout))
	}
	if c.Server.Timeout > 300 {
		errs = append(errs, fmt.Sprintf("SERVER_TIMEOUT should not exceed 300 seconds, got %d", c.Server.Timeout))
	}

	// 验证MaxConns
	if c.Server.MaxConns <= 0 {
		errs = append(errs, fmt.Sprintf("MAX_CONNECTIONS must be greater than 0, got %d", c.Server.MaxConns))
	}
	if c.Server.MaxConns > 10000 {
		errs = append(errs, fmt.Sprintf("MAX_CONNECTIONS should not exceed 10000, got %d", c.Server.MaxConns))
	}

	// 验证LogLevel
	validLogLevels := map[string]bool{
		"debug": true, "info": true, "warn": true, "error": true,
	}
	if !validLogLevels[strings.ToLower(c.Server.LogLevel)] {
		errs = append(errs, fmt.Sprintf("LOG_LEVEL must be one of [debug, info, warn, error], got %s", c.Server.LogLevel))
	}

	if len(errs) > 0 {
		return fmt.Errorf("configuration validation failed: %s", strings.Join(errs, "; "))
	}

	return nil
}

// validatePort 验证端口号是否在有效范围内
func validatePort(portStr, portName string) error {
	// 端口不能为空
	if portStr == "" {
		return fmt.Errorf("%s cannot be empty", portName)
	}

	// 端口必须是数字
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("%s must be a valid number, got %s", portName, portStr)
	}

	// 端口范围验证
	if port < MinPort || port > MaxPort {
		return fmt.Errorf("%s must be between %d and %d, got %d", portName, MinPort, MaxPort, port)
	}

	return nil
}

// GetGRPCAddr 获取gRPC地址
func (c *Config) GetGRPCAddr() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return fmt.Sprintf(":%s", c.Server.GRPCPort)
}

// GetHTTPAddr 获取HTTP地址
func (c *Config) GetHTTPAddr() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return fmt.Sprintf(":%s", c.Server.HTTPPort)
}

// GetTimeout 获取超时时间
func (c *Config) GetTimeout() time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return time.Duration(c.Server.Timeout) * time.Second
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string) bool {
	value := os.Getenv(key)
	switch strings.ToLower(value) {
	case "true", "1", "yes", "on":
		return true
	default:
		return false
	}
}

func getEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
