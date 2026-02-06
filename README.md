# gRPC-Hello (重构版)

生产就绪的gRPC微服务，带HTTP/JSON网关、多语言支持、统计跟踪和全面监控。

## 🏗️ 重构架构

```
grpc-hello/
├── main.go                    # 应用入口
├── api/dto/                   # 数据传输对象
│   ├── response.go           # 统一响应格式
│   └── error.go              # 错误码定义
├── internal/                  # 内部模块
│   ├── config/               # 配置管理
│   │   └── config.go        # 配置加载和验证
│   ├── handler/              # 处理器层
│   │   ├── grpc.go          # gRPC处理器
│   │   ├── http.go          # HTTP处理器
│   │   └── errors.go        # 错误处理
│   ├── middleware/           # 中间件
│   │   └── common.go        # 日志、追踪、CORS等
│   ├── service/             # 业务逻辑层
│   │   ├── greeting.go      # 问候服务
│   │   └── greeting_test.go # 测试用例
│   └── server/              # 服务器封装
│       └── server.go        # gRPC/HTTP服务启动
├── proto/                    # Protocol Buffers
│   └── helloworld/
├── route/                    # 路由扩展
│   └── route.go
├── client/                   # gRPC客户端示例
├── Makefile                 # 构建脚本
├── Dockerfile               # 容器配置
└── go.mod
```

## ✨ 重构亮点

### 1. 标准分层架构
- **Handler层**：处理HTTP/gRPC请求
- **Service层**：业务逻辑解耦
- **Middleware层**：统一中间件
- **DTO层**：请求/响应标准化

### 2. 统一错误处理
```go
// 错误码定义
const (
    CodeSuccess       = 0
    CodeBadRequest    = 400
    CodeTooManyNames  = 6001
)

// 使用示例
return nil, NewBusinessError(CodeTooManyNames, "too many names")
```

### 3. 统一响应格式
```go
// 所有API返回统一格式
{
    "code": 0,
    "message": "success",
    "data": {...},
    "time": 1234567890
}
```

### 4. 中间件支持
- 请求ID追踪
- 日志记录
- 恢复保护
- CORS跨域
- 请求超时

### 5. 配置验证
```go
// 启动时验证配置
if err := cfg.Validate(); err != nil {
    log.Fatalf("Configuration error: %v", err)
}
```

## 🚀 快速开始

```bash
# 安装依赖
make deps

# 构建项目
make build

# 运行服务
make run
```

## ⚙️ 配置

| 环境变量 | 描述 | 默认值 |
|---------|------|--------|
| GRPC_PORT | gRPC端口 | 8080 |
| HTTP_PORT | HTTP端口 | 8090 |
| ENABLE_DEBUG | 调试模式 | false |
| SERVER_TIMEOUT | 超时时间(秒) | 30 |
| ENABLE_REFLECTION | gRPC反射 | false |
| ENABLE_STATS | 统计功能 | true |

## 📡 API端点

- **健康检查**: `GET /health`
- **指标**: `GET /metrics`
- **问候**: `POST /rpc/v1/sayHello`
- **批量问候**: `POST /rpc/v1/sayHelloMultiple`
- **统计**: `GET /rpc/v1/greetingStats`

## 🧪 测试

```bash
# 运行所有测试
make test

# 运行特定包测试
go test ./internal/service/ -v
```

## 🐳 Docker

```bash
# 构建镜像
make docker-build

# 运行容器
make docker-run
```

## 📊 监控

- Prometheus指标: `/metrics`
- 健康检查: `/health`
- 就绪检查: `/ready`
- 存活检查: `/live`

---

## 📝 更新日志

### v2.0.0 (2026-02-06)
- ✨ 重构为标准分层架构
- ➕ 新增统一响应格式和错误码
- ➕ 新增请求追踪ID
- ➕ 新增CORS中间件
- ➕ 新增配置验证
- 🐛 修复endless库弃用问题
- ✅ 新增服务层单元测试
