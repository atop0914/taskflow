package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/spf13/viper"
)

// 端口范围常量
const (
	MinPort = 1
	MaxPort = 65535
)

// 默认配置常量
const (
	// Server defaults
	DefaultGRPCPort     = "9000"
	DefaultHTTPPort     = "9001"
	DefaultDBPath       = "~/.taskflow/taskflow.db"
	DefaultTimeout      = 30  // seconds
	DefaultMaxConns     = 1000
	DefaultLogLevel     = "info"
	DefaultMaxGreetings = 100

	// Worker defaults
	DefaultWorkerCount    = 4
	DefaultWorkerQueueSize = 1000
	DefaultWorkerRetryMax  = 3
	DefaultWorkerRetryDelay = 5 // seconds

	// Queue defaults
	DefaultQueueName    = "default"
	DefaultQueuePrefetch = 10
	DefaultQueueTimeout = 300 // seconds

	// Database defaults
	DefaultDBHost         = "localhost"
	DefaultDBPort         = "5432"
	DefaultDBName         = "taskflow"
	DefaultDBMaxOpenConns = 25
	DefaultDBMaxIdleConns = 5
	DefaultDBConnMaxLifetime = 300 // seconds
)

// ServerConfig 服务配置
//goland:noinspection GoDeprecation
type ServerConfig struct {
	GRPCPort    string `yaml:"grpc_port" env:"GRPC_PORT"`       // gRPC服务端口 (1-65535)
	HTTPPort    string `yaml:"http_port" env:"HTTP_PORT"`       // HTTP服务端口 (1-65535)
	DBPath      string `yaml:"db_path" env:"TASKFLOW_DB_PATH"`  // 数据库文件路径
	EnableDebug bool   `yaml:"enable_debug" env:"ENABLE_DEBUG"` // 启用调试模式
	Timeout     int    `yaml:"timeout" env:"SERVER_TIMEOUT"`     // 请求超时时间（秒），默认30秒
	MaxConns    int    `yaml:"max_conns" env:"MAX_CONNECTIONS"` // 最大连接数，默认1000
	LogLevel    string `yaml:"log_level" env:"LOG_LEVEL"`       // 日志级别：debug, info, warn, error
}

// FeatureFlags 功能开关
type FeatureFlags struct {
	EnableReflection bool `yaml:"enable_reflection" env:"ENABLE_REFLECTION"` // 启用gRPC反射
	EnableStats      bool `yaml:"enable_stats" env:"ENABLE_STATS"`          // 启用统计功能
	EnableMetrics    bool `yaml:"enable_metrics" env:"METRICS_ENABLED"`     // 启用Prometheus指标
	MaxGreetings     int  `yaml:"max_greetings" env:"MAX_GREETINGS"`        // 最大问候数量，默认100
}

// WorkerConfig Worker配置
type WorkerConfig struct {
	Count       int    `yaml:"count" env:"WORKER_COUNT"`                     // Worker数量，默认4
	QueueSize   int    `yaml:"queue_size" env:"WORKER_QUEUE_SIZE"`           // 每个Worker的队列大小，默认1000
	RetryMax    int    `yaml:"retry_max" env:"WORKER_RETRY_MAX"`             // 最大重试次数，默认3
	RetryDelay  int    `yaml:"retry_delay" env:"WORKER_RETRY_DELAY"`         // 重试延迟（秒），默认5
	Timeout     int    `yaml:"timeout" env:"WORKER_TIMEOUT"`                  // Worker执行超时（秒），默认300
	BatchSize   int    `yaml:"batch_size" env:"WORKER_BATCH_SIZE"`           // 批处理大小，默认10
	AutoScale   bool   `yaml:"auto_scale" env:"WORKER_AUTO_SCALE"`          // 是否自动扩缩容
	MinScale    int    `yaml:"min_scale" env:"WORKER_MIN_SCALE"`             // 最小Worker数量
	MaxScale    int    `yaml:"max_scale" env:"WORKER_MAX_SCALE"`             // 最大Worker数量
	Heartbeat   int    `yaml:"heartbeat" env:"WORKER_HEARTBEAT"`             // 心跳间隔（秒），默认30
}

// QueueConfig Queue配置
type QueueConfig struct {
	Name           string `yaml:"name" env:"QUEUE_NAME"`                           // 队列名称，默认default
	Prefetch       int    `yaml:"prefetch" env:"QUEUE_PREFETCH"`                   // 预取数量，默认10
	Timeout        int    `yaml:"timeout" env:"QUEUE_TIMEOUT"`                      // 队列超时（秒），默认300
	MaxLength      int    `yaml:"max_length" env:"QUEUE_MAX_LENGTH"`               // 队列最大长度，0表示无限制
	Priority       int    `yaml:"priority" env:"QUEUE_PRIORITY"`                   // 队列优先级，0-10，默认5
	Durable        bool   `yaml:"durable" env:"QUEUE_DURABLE"`                     // 是否持久化
	AutoDelete     bool   `yaml:"auto_delete" env:"QUEUE_AUTO_DELETE"`             // 是否自动删除
	Exchange       string `yaml:"exchange" env:"QUEUE_EXCHANGE"`                   // 交换机名称
	RoutingKey     string `yaml:"routing_key" env:"QUEUE_ROUTING_KEY"`             // 路由键
	DeadLetterExchange string `yaml:"dead_letter_exchange" env:"QUEUE_DLX"`         // 死信交换机
	DeadLetterQueue    string `yaml:"dead_letter_queue" env:"QUEUE_DLQ"`           // 死信队列
	TTL            int    `yaml:"ttl" env:"QUEUE_TTL"`                             // 消息TTL（毫秒）
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host            string `yaml:"host" env:"DB_HOST"`                       // 数据库主机，默认localhost
	Port            string `yaml:"port" env:"DB_PORT"`                       // 数据库端口，默认5432
	Name            string `yaml:"name" env:"DB_NAME"`                       // 数据库名称，默认taskflow
	User            string `yaml:"user" env:"DB_USER"`                       // 数据库用户
	Password        string `yaml:"password" env:"DB_PASSWORD"`               // 数据库密码
	SSLMode         string `yaml:"ssl_mode" env:"DB_SSL_MODE"`               // SSL模式，默认disable
	MaxOpenConns    int    `yaml:"max_open_conns" env:"DB_MAX_OPEN_CONNS"`    // 最大打开连接数，默认25
	MaxIdleConns    int    `yaml:"max_idle_conns" env:"DB_MAX_IDLE_CONNS"`    // 最大空闲连接数，默认5
	ConnMaxLifetime int    `yaml:"conn_max_lifetime" env:"DB_CONN_MAX_LIFETIME"` // 连接最大生命周期（秒），默认300
	ConnMaxIdleTime int    `yaml:"conn_max_idle_time" env:"DB_CONN_MAX_IDLE_TIME"` // 空闲连接最大时间（秒），默认60
	MaxRetries      int    `yaml:"max_retries" env:"DB_MAX_RETRIES"`          // 最大重试次数，默认3
	RetryDelay      int    `yaml:"retry_delay" env:"DB_RETRY_DELAY"`          // 重试延迟（毫秒），默认100
	TablePrefix     string `yaml:"table_prefix" env:"DB_TABLE_PREFIX"`        // 表前缀，默认空
	PoolSize        int    `yaml:"pool_size" env:"DB_POOL_SIZE"`              // 连接池大小
	MinIdleConns    int    `yaml:"min_idle_conns" env:"DB_MIN_IDLE_CONNS"`    // 最小空闲连接数
}

// Config 配置
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Features FeatureFlags   `yaml:"features"`
	Worker   WorkerConfig   `yaml:"worker"`
	Queue    QueueConfig    `yaml:"queue"`
	Database DatabaseConfig `yaml:"database"`
	mu       sync.RWMutex   // 用于配置热加载
}

// LoadConfig 加载配置（支持环境变量覆盖）
// 环境变量优先级高于配置文件默认值
// InitViper 初始化 Viper 配置（支持 .env 和 config.yaml）
func InitViper() *viper.Viper {
	v := viper.New()

	// 设置配置文件名
	v.SetConfigName("config")
	v.SetConfigType("yaml")

	// 添加配置文件路径
	v.AddConfigPath(".")
	v.AddConfigPath("$HOME/.taskflow")
	v.AddConfigPath("/etc/taskflow")

	// 优先从环境变量读取（环境变量优先级最高）
	// TASKFLOW_GRPC_ADDR 和 TASKFLOW_HTTP_ADDR 格式为 ":port"
	v.SetEnvPrefix("TASKFLOW")
	v.BindEnv("GRPC_ADDR")
	v.BindEnv("HTTP_ADDR")
	v.BindEnv("DB_PATH")

	// 自动加载环境变量（支持 .env 文件）
	v.AutomaticEnv()

	// 尝试读取配置文件（如果存在）
	_ = v.ReadInConfig()

	return v
}

// LoadConfig 加载配置（支持环境变量覆盖）
// 环境变量优先级高于配置文件默认值
func LoadConfig() *Config {
	// 初始化 viper
	v := InitViper()

	// 从环境变量或 viper 获取配置
	grpcAddr := getEnv("TASKFLOW_GRPC_ADDR", "")
	httpAddr := getEnv("TASKFLOW_HTTP_ADDR", "")
	dbPath := getEnv("TASKFLOW_DB_PATH", "")

	// 解析地址获取端口
	grpcPort := DefaultGRPCPort
	httpPort := DefaultHTTPPort

	if grpcAddr != "" {
		grpcPort = strings.TrimPrefix(grpcAddr, ":")
	}
	if httpAddr != "" {
		httpPort = strings.TrimPrefix(httpAddr, ":")
	}

	// 如果 viper 中有配置，使用 viper 的值
	if v.IsSet("server.grpc_port") {
		grpcPort = v.GetString("server.grpc_port")
	}
	if v.IsSet("server.http_port") {
		httpPort = v.GetString("server.http_port")
	}
	if v.IsSet("server.db_path") {
		dbPath = v.GetString("server.db_path")
	}

	// 如果没有设置 DBPath，使用默认值
	if dbPath == "" {
		dbPath = DefaultDBPath
	}

	cfg := &Config{
		Server: ServerConfig{
			GRPCPort:    grpcPort,
			HTTPPort:    httpPort,
			DBPath:      dbPath,
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
		Worker: WorkerConfig{
			Count:       getEnvInt("WORKER_COUNT", DefaultWorkerCount),
			QueueSize:   getEnvInt("WORKER_QUEUE_SIZE", DefaultWorkerQueueSize),
			RetryMax:    getEnvInt("WORKER_RETRY_MAX", DefaultWorkerRetryMax),
			RetryDelay:  getEnvInt("WORKER_RETRY_DELAY", DefaultWorkerRetryDelay),
			Timeout:     getEnvInt("WORKER_TIMEOUT", DefaultQueueTimeout),
			BatchSize:   getEnvInt("WORKER_BATCH_SIZE", 10),
			AutoScale:   getEnvBool("WORKER_AUTO_SCALE"),
			MinScale:    getEnvInt("WORKER_MIN_SCALE", DefaultWorkerCount),
			MaxScale:    getEnvInt("WORKER_MAX_SCALE", DefaultWorkerCount*2),
			Heartbeat:   getEnvInt("WORKER_HEARTBEAT", 30),
		},
		Queue: QueueConfig{
			Name:               getEnv("QUEUE_NAME", DefaultQueueName),
			Prefetch:           getEnvInt("QUEUE_PREFETCH", DefaultQueuePrefetch),
			Timeout:            getEnvInt("QUEUE_TIMEOUT", DefaultQueueTimeout),
			MaxLength:          getEnvInt("QUEUE_MAX_LENGTH", 0),
			Priority:           getEnvInt("QUEUE_PRIORITY", 5),
			Durable:            getEnvBool("QUEUE_DURABLE"),
			AutoDelete:         getEnvBool("QUEUE_AUTO_DELETE"),
			Exchange:           getEnv("QUEUE_EXCHANGE", ""),
			RoutingKey:         getEnv("QUEUE_ROUTING_KEY", ""),
			DeadLetterExchange: getEnv("QUEUE_DLX", ""),
			DeadLetterQueue:    getEnv("QUEUE_DLQ", ""),
			TTL:                getEnvInt("QUEUE_TTL", 0),
		},
		Database: DatabaseConfig{
			Host:             getEnv("DB_HOST", DefaultDBHost),
			Port:             getEnv("DB_PORT", DefaultDBPort),
			Name:             getEnv("DB_NAME", DefaultDBName),
			User:             getEnv("DB_USER", ""),
			Password:         getEnv("DB_PASSWORD", ""),
			SSLMode:          getEnv("DB_SSL_MODE", "disable"),
			MaxOpenConns:     getEnvInt("DB_MAX_OPEN_CONNS", DefaultDBMaxOpenConns),
			MaxIdleConns:     getEnvInt("DB_MAX_IDLE_CONNS", DefaultDBMaxIdleConns),
			ConnMaxLifetime:  getEnvInt("DB_CONN_MAX_LIFETIME", DefaultDBConnMaxLifetime),
			ConnMaxIdleTime:  getEnvInt("DB_CONN_MAX_IDLE_TIME", 60),
			MaxRetries:       getEnvInt("DB_MAX_RETRIES", 3),
			RetryDelay:       getEnvInt("DB_RETRY_DELAY", 100),
			TablePrefix:      getEnv("DB_TABLE_PREFIX", ""),
			PoolSize:         getEnvInt("DB_POOL_SIZE", DefaultDBMaxOpenConns),
			MinIdleConns:     getEnvInt("DB_MIN_IDLE_CONNS", DefaultDBMaxIdleConns),
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

	// 验证Worker配置
	if c.Worker.Count <= 0 {
		errs = append(errs, fmt.Sprintf("WORKER_COUNT must be greater than 0, got %d", c.Worker.Count))
	}
	if c.Worker.Count > 100 {
		errs = append(errs, fmt.Sprintf("WORKER_COUNT should not exceed 100, got %d", c.Worker.Count))
	}
	if c.Worker.QueueSize <= 0 {
		errs = append(errs, fmt.Sprintf("WORKER_QUEUE_SIZE must be greater than 0, got %d", c.Worker.QueueSize))
	}
	if c.Worker.RetryMax < 0 {
		errs = append(errs, fmt.Sprintf("WORKER_RETRY_MAX must be non-negative, got %d", c.Worker.RetryMax))
	}
	if c.Worker.Timeout <= 0 {
		errs = append(errs, fmt.Sprintf("WORKER_TIMEOUT must be greater than 0, got %d", c.Worker.Timeout))
	}
	if c.Worker.AutoScale {
		if c.Worker.MinScale <= 0 {
			errs = append(errs, fmt.Sprintf("WORKER_MIN_SCALE must be greater than 0 when auto_scale is enabled, got %d", c.Worker.MinScale))
		}
		if c.Worker.MaxScale < c.Worker.MinScale {
			errs = append(errs, fmt.Sprintf("WORKER_MAX_SCALE (%d) must be greater than or equal to WORKER_MIN_SCALE (%d)", c.Worker.MaxScale, c.Worker.MinScale))
		}
	}

	// 验证Queue配置
	if c.Queue.Name == "" {
		errs = append(errs, "QUEUE_NAME cannot be empty")
	}
	if c.Queue.Prefetch < 0 {
		errs = append(errs, fmt.Sprintf("QUEUE_PREFETCH must be non-negative, got %d", c.Queue.Prefetch))
	}
	if c.Queue.Timeout <= 0 {
		errs = append(errs, fmt.Sprintf("QUEUE_TIMEOUT must be greater than 0, got %d", c.Queue.Timeout))
	}
	if c.Queue.MaxLength < 0 {
		errs = append(errs, fmt.Sprintf("QUEUE_MAX_LENGTH must be non-negative, got %d", c.Queue.MaxLength))
	}
	if c.Queue.Priority < 0 || c.Queue.Priority > 10 {
		errs = append(errs, fmt.Sprintf("QUEUE_PRIORITY must be between 0 and 10, got %d", c.Queue.Priority))
	}
	if c.Queue.TTL < 0 {
		errs = append(errs, fmt.Sprintf("QUEUE_TTL must be non-negative, got %d", c.Queue.TTL))
	}

	// 验证Database配置
	if c.Database.Host == "" {
		errs = append(errs, "DB_HOST cannot be empty")
	}
	if err := validatePort(c.Database.Port, "DB_PORT"); err != nil {
		errs = append(errs, err.Error())
	}
	if c.Database.Name == "" {
		errs = append(errs, "DB_NAME cannot be empty")
	}
	if c.Database.MaxOpenConns <= 0 {
		errs = append(errs, fmt.Sprintf("DB_MAX_OPEN_CONNS must be greater than 0, got %d", c.Database.MaxOpenConns))
	}
	if c.Database.MaxOpenConns > 1000 {
		errs = append(errs, fmt.Sprintf("DB_MAX_OPEN_CONNS should not exceed 1000, got %d", c.Database.MaxOpenConns))
	}
	if c.Database.MaxIdleConns < 0 {
		errs = append(errs, fmt.Sprintf("DB_MAX_IDLE_CONNS must be non-negative, got %d", c.Database.MaxIdleConns))
	}
	if c.Database.MaxIdleConns > c.Database.MaxOpenConns {
		errs = append(errs, fmt.Sprintf("DB_MAX_IDLE_CONNS (%d) cannot exceed DB_MAX_OPEN_CONNS (%d)", c.Database.MaxIdleConns, c.Database.MaxOpenConns))
	}
	if c.Database.ConnMaxLifetime <= 0 {
		errs = append(errs, fmt.Sprintf("DB_CONN_MAX_LIFETIME must be greater than 0, got %d", c.Database.ConnMaxLifetime))
	}
	if c.Database.MaxRetries < 0 {
		errs = append(errs, fmt.Sprintf("DB_MAX_RETRIES must be non-negative, got %d", c.Database.MaxRetries))
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

// ValidateWorker 验证Worker配置（独立方法）
func (c *Config) ValidateWorker() error {
	return c.Worker.Validate()
}

// ValidateWorker 验证Worker配置
func (w *WorkerConfig) Validate() error {
	var errs []string

	if w.Count <= 0 {
		errs = append(errs, fmt.Sprintf("WORKER_COUNT must be greater than 0, got %d", w.Count))
	}
	if w.Count > 100 {
		errs = append(errs, fmt.Sprintf("WORKER_COUNT should not exceed 100, got %d", w.Count))
	}

	if w.QueueSize <= 0 {
		errs = append(errs, fmt.Sprintf("WORKER_QUEUE_SIZE must be greater than 0, got %d", w.QueueSize))
	}
	if w.QueueSize > 100000 {
		errs = append(errs, fmt.Sprintf("WORKER_QUEUE_SIZE should not exceed 100000, got %d", w.QueueSize))
	}

	if w.RetryMax < 0 {
		errs = append(errs, fmt.Sprintf("WORKER_RETRY_MAX must be non-negative, got %d", w.RetryMax))
	}
	if w.RetryMax > 100 {
		errs = append(errs, fmt.Sprintf("WORKER_RETRY_MAX should not exceed 100, got %d", w.RetryMax))
	}

	if w.RetryDelay < 0 {
		errs = append(errs, fmt.Sprintf("WORKER_RETRY_DELAY must be non-negative, got %d", w.RetryDelay))
	}
	if w.RetryDelay > 3600 {
		errs = append(errs, fmt.Sprintf("WORKER_RETRY_DELAY should not exceed 3600 seconds, got %d", w.RetryDelay))
	}

	if w.Timeout <= 0 {
		errs = append(errs, fmt.Sprintf("WORKER_TIMEOUT must be greater than 0, got %d", w.Timeout))
	}
	if w.Timeout > 86400 {
		errs = append(errs, fmt.Sprintf("WORKER_TIMEOUT should not exceed 86400 seconds (24h), got %d", w.Timeout))
	}

	if w.BatchSize <= 0 {
		errs = append(errs, fmt.Sprintf("WORKER_BATCH_SIZE must be greater than 0, got %d", w.BatchSize))
	}
	if w.BatchSize > 1000 {
		errs = append(errs, fmt.Sprintf("WORKER_BATCH_SIZE should not exceed 1000, got %d", w.BatchSize))
	}

	if w.AutoScale {
		if w.MinScale <= 0 {
			errs = append(errs, fmt.Sprintf("WORKER_MIN_SCALE must be greater than 0 when auto-scaling, got %d", w.MinScale))
		}
		if w.MaxScale < w.MinScale {
			errs = append(errs, fmt.Sprintf("WORKER_MAX_SCALE (%d) must be greater than or equal to WORKER_MIN_SCALE (%d)", w.MaxScale, w.MinScale))
		}
	}

	if w.Heartbeat <= 0 {
		errs = append(errs, fmt.Sprintf("WORKER_HEARTBEAT must be greater than 0, got %d", w.Heartbeat))
	}
	if w.Heartbeat > 300 {
		errs = append(errs, fmt.Sprintf("WORKER_HEARTBEAT should not exceed 300 seconds, got %d", w.Heartbeat))
	}

	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}

// ValidateQueue 验证Queue配置（独立方法）
func (c *Config) ValidateQueue() error {
	return c.Queue.Validate()
}

// Validate 验证Queue配置
func (q *QueueConfig) Validate() error {
	var errs []string

	if q.Name == "" {
		errs = append(errs, "QUEUE_NAME cannot be empty")
	}
	if len(q.Name) > 255 {
		errs = append(errs, fmt.Sprintf("QUEUE_NAME should not exceed 255 characters, got %d", len(q.Name)))
	}

	if q.Prefetch <= 0 {
		errs = append(errs, fmt.Sprintf("QUEUE_PREFETCH must be greater than 0, got %d", q.Prefetch))
	}
	if q.Prefetch > 1000 {
		errs = append(errs, fmt.Sprintf("QUEUE_PREFETCH should not exceed 1000, got %d", q.Prefetch))
	}

	if q.Timeout <= 0 {
		errs = append(errs, fmt.Sprintf("QUEUE_TIMEOUT must be greater than 0, got %d", q.Timeout))
	}
	if q.Timeout > 86400 {
		errs = append(errs, fmt.Sprintf("QUEUE_TIMEOUT should not exceed 86400 seconds (24h), got %d", q.Timeout))
	}

	if q.MaxLength < 0 {
		errs = append(errs, fmt.Sprintf("QUEUE_MAX_LENGTH must be non-negative, got %d", q.MaxLength))
	}

	if q.Priority < 0 || q.Priority > 10 {
		errs = append(errs, fmt.Sprintf("QUEUE_PRIORITY must be between 0 and 10, got %d", q.Priority))
	}

	if q.TTL < 0 {
		errs = append(errs, fmt.Sprintf("QUEUE_TTL must be non-negative, got %d", q.TTL))
	}
	if q.TTL > 604800000 { // 7 days in milliseconds
		errs = append(errs, fmt.Sprintf("QUEUE_TTL should not exceed 604800000 ms (7 days), got %d", q.TTL))
	}

	if q.Exchange != "" && len(q.Exchange) > 255 {
		errs = append(errs, fmt.Sprintf("QUEUE_EXCHANGE should not exceed 255 characters, got %d", len(q.Exchange)))
	}

	if q.RoutingKey != "" && len(q.RoutingKey) > 255 {
		errs = append(errs, fmt.Sprintf("QUEUE_ROUTING_KEY should not exceed 255 characters, got %d", len(q.RoutingKey)))
	}

	if q.DeadLetterExchange != "" && len(q.DeadLetterExchange) > 255 {
		errs = append(errs, fmt.Sprintf("QUEUE_DLX should not exceed 255 characters, got %d", len(q.DeadLetterExchange)))
	}

	if q.DeadLetterQueue != "" && len(q.DeadLetterQueue) > 255 {
		errs = append(errs, fmt.Sprintf("QUEUE_DLQ should not exceed 255 characters, got %d", len(q.DeadLetterQueue)))
	}

	if (q.DeadLetterExchange == "" && q.DeadLetterQueue != "") || (q.DeadLetterExchange != "" && q.DeadLetterQueue == "") {
		errs = append(errs, "QUEUE_DLX and QUEUE_DLQ must both be set or both be empty")
	}

	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}

// ValidateDatabase 验证Database配置（独立方法）
func (c *Config) ValidateDatabase() error {
	return c.Database.Validate()
}

// Validate 验证Database配置
func (d *DatabaseConfig) Validate() error {
	var errs []string

	// 验证Host
	if d.Host == "" {
		errs = append(errs, "DB_HOST cannot be empty")
	}
	if len(d.Host) > 255 {
		errs = append(errs, fmt.Sprintf("DB_HOST should not exceed 255 characters, got %d", len(d.Host)))
	}

	// 验证Port
	if err := validatePort(d.Port, "DB_PORT"); err != nil {
		errs = append(errs, err.Error())
	}

	// 验证Name
	if d.Name == "" {
		errs = append(errs, "DB_NAME cannot be empty")
	}
	if len(d.Name) > 63 {
		errs = append(errs, fmt.Sprintf("DB_NAME should not exceed 63 characters, got %d", len(d.Name)))
	}

	// 验证User（可选，但如果提供则验证）
	if d.User != "" && len(d.User) > 63 {
		errs = append(errs, fmt.Sprintf("DB_USER should not exceed 63 characters, got %d", len(d.User)))
	}

	// 验证SSLMode
	validSSLModes := map[string]bool{
		"disable": true, "require": true, "verify-full": true, "allow": true, "prefer": true,
	}
	if !validSSLModes[d.SSLMode] {
		errs = append(errs, fmt.Sprintf("DB_SSL_MODE must be one of [disable, require, verify-full, allow, prefer], got %s", d.SSLMode))
	}

	// 验证连接池配置
	if d.MaxOpenConns <= 0 {
		errs = append(errs, fmt.Sprintf("DB_MAX_OPEN_CONNS must be greater than 0, got %d", d.MaxOpenConns))
	}
	if d.MaxOpenConns > 1000 {
		errs = append(errs, fmt.Sprintf("DB_MAX_OPEN_CONNS should not exceed 1000, got %d", d.MaxOpenConns))
	}

	if d.MaxIdleConns < 0 {
		errs = append(errs, fmt.Sprintf("DB_MAX_IDLE_CONNS must be non-negative, got %d", d.MaxIdleConns))
	}
	if d.MaxIdleConns > d.MaxOpenConns {
		errs = append(errs, fmt.Sprintf("DB_MAX_IDLE_CONNS (%d) cannot exceed DB_MAX_OPEN_CONNS (%d)", d.MaxIdleConns, d.MaxOpenConns))
	}

	if d.ConnMaxLifetime <= 0 {
		errs = append(errs, fmt.Sprintf("DB_CONN_MAX_LIFETIME must be greater than 0, got %d", d.ConnMaxLifetime))
	}
	if d.ConnMaxLifetime > 3600 {
		errs = append(errs, fmt.Sprintf("DB_CONN_MAX_LIFETIME should not exceed 3600 seconds (1h), got %d", d.ConnMaxLifetime))
	}

	if d.ConnMaxIdleTime < 0 {
		errs = append(errs, fmt.Sprintf("DB_CONN_MAX_IDLE_TIME must be non-negative, got %d", d.ConnMaxIdleTime))
	}
	if d.ConnMaxIdleTime > 3600 {
		errs = append(errs, fmt.Sprintf("DB_CONN_MAX_IDLE_TIME should not exceed 3600 seconds (1h), got %d", d.ConnMaxIdleTime))
	}

	if d.MaxRetries < 0 {
		errs = append(errs, fmt.Sprintf("DB_MAX_RETRIES must be non-negative, got %d", d.MaxRetries))
	}
	if d.MaxRetries > 100 {
		errs = append(errs, fmt.Sprintf("DB_MAX_RETRIES should not exceed 100, got %d", d.MaxRetries))
	}

	if d.RetryDelay < 0 {
		errs = append(errs, fmt.Sprintf("DB_RETRY_DELAY must be non-negative, got %d", d.RetryDelay))
	}
	if d.RetryDelay > 60000 {
		errs = append(errs, fmt.Sprintf("DB_RETRY_DELAY should not exceed 60000 ms (1m), got %d", d.RetryDelay))
	}

	if d.TablePrefix != "" && len(d.TablePrefix) > 16 {
		errs = append(errs, fmt.Sprintf("DB_TABLE_PREFIX should not exceed 16 characters, got %d", len(d.TablePrefix)))
	}

	if d.PoolSize <= 0 {
		errs = append(errs, fmt.Sprintf("DB_POOL_SIZE must be greater than 0, got %d", d.PoolSize))
	}
	if d.PoolSize > d.MaxOpenConns {
		errs = append(errs, fmt.Sprintf("DB_POOL_SIZE (%d) cannot exceed DB_MAX_OPEN_CONNS (%d)", d.PoolSize, d.MaxOpenConns))
	}

	if d.MinIdleConns < 0 {
		errs = append(errs, fmt.Sprintf("DB_MIN_IDLE_CONNS must be non-negative, got %d", d.MinIdleConns))
	}
	if d.MinIdleConns > d.PoolSize {
		errs = append(errs, fmt.Sprintf("DB_MIN_IDLE_CONNS (%d) cannot exceed DB_POOL_SIZE (%d)", d.MinIdleConns, d.PoolSize))
	}

	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}

// GetWorkerTimeout 获取Worker超时时间
func (c *Config) GetWorkerTimeout() time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return time.Duration(c.Worker.Timeout) * time.Second
}

// GetWorkerRetryDelay 获取Worker重试延迟
func (c *Config) GetWorkerRetryDelay() time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return time.Duration(c.Worker.RetryDelay) * time.Second
}

// GetQueueTimeout 获取队列超时时间
func (c *Config) GetQueueTimeout() time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return time.Duration(c.Queue.Timeout) * time.Second
}

// GetQueueTTL 获取消息TTL
func (c *Config) GetQueueTTL() time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return time.Duration(c.Queue.TTL) * time.Millisecond
}

// GetDBConnMaxLifetime 获取数据库连接最大生命周期
func (c *Config) GetDBConnMaxLifetime() time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return time.Duration(c.Database.ConnMaxLifetime) * time.Second
}

// GetDBConnMaxIdleTime 获取数据库空闲连接最大时间
func (c *Config) GetDBConnMaxIdleTime() time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return time.Duration(c.Database.ConnMaxIdleTime) * time.Second
}

// GetDBRetryDelay 获取数据库重试延迟
func (c *Config) GetDBRetryDelay() time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return time.Duration(c.Database.RetryDelay) * time.Millisecond
}

// GetDSN 获取数据库连接字符串
func (c *Config) GetDSN() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return fmt.Sprintf(
		"host=%s port=%s dbname=%s user=%s password=%s sslmode=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.Name,
		c.Database.User,
		c.Database.Password,
		c.Database.SSLMode,
	)
}
