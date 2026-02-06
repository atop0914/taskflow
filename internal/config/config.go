package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ServerConfig 服务配置
type ServerConfig struct {
	GRPCPort    string `yaml:"grpc_port" env:"GRPC_PORT"`
	HTTPPort    string `yaml:"http_port" env:"HTTP_PORT"`
	EnableDebug bool   `yaml:"enable_debug" env:"ENABLE_DEBUG"`
	Timeout     int    `yaml:"timeout" env:"SERVER_TIMEOUT"`
	MaxConns    int    `yaml:"max_conns" env:"MAX_CONNECTIONS"`
	LogLevel    string `yaml:"log_level" env:"LOG_LEVEL"`
}

// FeatureFlags 功能开关
type FeatureFlags struct {
	EnableReflection bool `yaml:"enable_reflection" env:"ENABLE_REFLECTION"`
	EnableStats      bool `yaml:"enable_stats" env:"ENABLE_STATS"`
	EnableMetrics    bool `yaml:"enable_metrics" env:"METRICS_ENABLED"`
	MaxGreetings     int  `yaml:"max_greetings" env:"MAX_GREETINGS"`
}

// Config 配置
type Config struct {
	Server   ServerConfig  `yaml:"server"`
	Features FeatureFlags  `yaml:"features"`
	mu       sync.RWMutex  // 用于配置热加载
}

// LoadConfig 加载配置（支持环境变量覆盖）
func LoadConfig() *Config {
	cfg := &Config{
		Server: ServerConfig{
			GRPCPort:    getEnv("GRPC_PORT", "8080"),
			HTTPPort:    getEnv("HTTP_PORT", "8090"),
			EnableDebug: getEnvBool("ENABLE_DEBUG"),
			Timeout:     getEnvInt("SERVER_TIMEOUT", 30),
			MaxConns:    getEnvInt("MAX_CONNECTIONS", 1000),
			LogLevel:    getEnv("LOG_LEVEL", "info"),
		},
		Features: FeatureFlags{
			EnableReflection: getEnvBool("ENABLE_REFLECTION"),
			EnableStats:      getEnvBool("ENABLE_STATS"),
			EnableMetrics:    getEnvBool("METRICS_ENABLED"),
			MaxGreetings:     getEnvInt("MAX_GREETINGS", 100),
		},
	}
	return cfg
}

// Validate 验证配置
func (c *Config) Validate() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var errs []string

	if c.Server.GRPCPort == "" {
		errs = append(errs, "GRPC_PORT cannot be empty")
	}

	if c.Server.HTTPPort == "" {
		errs = append(errs, "HTTP_PORT cannot be empty")
	}

	if c.Features.MaxGreetings <= 0 {
		errs = append(errs, "MAX_GREETINGS must be greater than 0")
	}

	if c.Server.Timeout <= 0 {
		errs = append(errs, "SERVER_TIMEOUT must be greater than 0")
	}

	if c.Server.MaxConns <= 0 {
		errs = append(errs, "MAX_CONNECTIONS must be greater than 0")
	}

	if len(errs) > 0 {
		return fmt.Errorf("configuration validation failed: %s", strings.Join(errs, "; "))
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
