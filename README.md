# TaskFlow

gRPC 任务调度服务。

## 🚀 特性

- **四种 gRPC 通信模式**
  - Simple RPC：单次请求/响应
  - Server Stream：服务端推送
  - Client Stream：批量创建任务
  - Bidirectional Stream：实时双向通信

- **异步任务处理**
  - SQLite 持久化存储
  - 任务队列管理
  - 状态机控制
  - 自动重试机制

- **生产级特性**
  - JWT 认证
  - 限流控制
  - 请求日志
  - 配置热加载

## 🏗️ 架构

```
taskflow/
├── proto/
│   ├── task.proto           # 服务定义
│   ├── task.pb.go          # 生成的 Go 代码
│   ├── task_grpc.pb.go     # 生成的 gRPC 代码
│   └── task_stream.go      # 流式处理
├── internal/
│   ├── config/             # 配置管理 ✅ 已完成
│   ├── model/              # 数据模型 ✅ 已完成
│   ├── error/              # 错误码定义与处理 ✅ 已完成
│   ├── repository/         # SQLite 数据访问层 ✅ 已完成
│   ├── middleware/         # 中间件 ✅ 已完成
│   ├── handler/            # gRPC Handler ✅ 已完成
│   ├── server/             # 服务入口 ✅ 已完成
│   ├── service/            # 业务逻辑层 ✅ 已完成
│   │   ├── task_service.go  # 任务服务
│   │   ├── scheduler.go    # 任务调度器
│   │   └── state_machine.go # 状态机
├── cmd/
│   └── server/              # 服务入口
└── scripts/                 # 工具脚本
```

## 📦 技术栈

- **Go 1.21+**
- **gRPC** (Google Protocol Buffers)
- **SQLite** (持久化)
- **Zerolog** (日志)

## 🛠️ 快速开始

```bash
# 安装依赖
go mod tidy

# 生成 proto 文件 (需要 buf 和 protoc)
buf generate

# 构建项目
go build -o taskflow .

# 运行服务
./taskflow

# 运行测试
go test ./...
```

## ⚙️ 配置

通过环境变量配置：

| 环境变量 | 描述 | 默认值 |
|---------|------|--------|
| GRPC_PORT | gRPC 端口 | 8080 |
| HTTP_PORT | HTTP 端口 | 8090 |
| DB_HOST | 数据库主机 | localhost |
| DB_PORT | 数据库端口 | 5432 |
| DB_NAME | 数据库名称 | taskflow |
| WORKER_COUNT | Worker 数量 | 4 |
| MAX_RETRIES | 最大重试次数 | 3 |

## ✅ 已完成功能

### 1. Service 层 (internal/service/)

完整的业务逻辑层实现：

| 方法 | 描述 |
|------|------|
| `CreateTask` | 创建任务，支持依赖管理 |
| `GetTask` | 获取任务 |
| `UpdateTask` | 更新任务状态和结果 |
| `CancelTask` | 取消任务 |
| `RetryTask` | 重试失败任务 |
| `ListTasks` | 分页查询任务 |
| `SearchTasks` | 关键词搜索 |
| `GetTaskEvents` | 获取任务事件 |
| `StartScheduler` | 启动任务调度器 |
| `StopScheduler` | 停止任务调度器 |

### 2. 任务调度器 (internal/service/scheduler.go)

| 功能 | 描述 |
|------|------|
| `WorkerPool` | 并发工作池，支持任务并行执行 |
| `TrySchedule` | 依赖检查与任务调度 |
| `executeTask` | 任务执行逻辑 |
| `handleTaskSuccess` | 任务成功后处理 |
| `handleTaskFailure` | 任务失败重试处理 |
| `GetStatus` | 获取调度器状态 |

### 3. 状态机 (internal/service/state_machine.go)

| 方法 | 描述 |
|------|------|
| `CanTransition` | 检查状态转换是否有效 |
| `Transition` | 执行状态转换 |
| `IsTerminal` | 判断是否为终态 |
| `GetAllowedTransitions` | 获取允许的状态转换 |

**状态转换规则：**
- `PENDING` → `RUNNING`, `CANCELLED`
- `RUNNING` → `SUCCEEDED`, `FAILED`, `TIMEOUT`, `CANCELLED`
- `FAILED` → `PENDING` (重试), `CANCELLED`
- 终态 (`SUCCEEDED`, `CANCELLED`, `TIMEOUT`) 不可转换

### 4. SQLite 持久化层 (internal/repository/)

提供完整的 CRUD 操作：

| 方法 | 描述 |
|------|------|
| `Create` | 创建任务 |
| `GetByID` | 根据 ID 获取任务 |
| `Update` | 更新任务 |
| `Delete` | 删除任务 |
| `List` | 分页列出任务 |
| `ListByStatus` | 按状态列出任务 |
| `ListPending` | 列出待处理任务 |
| `ListByCreator` | 按创建者查询 |
| `ListByFilter` | 多条件过滤查询 |
| `Search` | 关键词搜索 |
| `Count` | 统计任务数量 |
| `UpdateStatus` | 更新任务状态 |
| `UpdateStatusWithEvent` | 原子更新+记录事件 |
| `AddEvent` | 添加任务事件 |
| `GetEventsByTaskID` | 获取任务所有事件 |

### 5. 错误处理模块 (internal/error/)

完整的错误码定义和错误处理函数：

**错误码定义：**
- 通用错误 (1xxx)：参数错误、未授权、禁止访问、未找到、超时等
- 任务相关错误 (2xxx)：任务未找到、运行中、终止/取消/超时、依赖未满足等
- 存储相关错误 (3xxx)：数据库错误、未连接、事务错误
- gRPC 相关错误 (4xxx)：服务未就绪、连接错误、超时

**错误处理函数：**
- `TaskError` 结构体实现 error 接口
- `HTTPStatusFromCode()` - 错误码转 HTTP 状态码
- `ToGRPCStatus()` / `FromGRPCStatus()` - gRPC status 互转
- `HandleGinError()` / `HandleGinErrorWithCode()` - 中间件错误处理
- `HandleGinPanic()` - Panic 恢复处理

### 6. 配置系统 (internal/config/)

完整的配置管理：
- 环境变量加载
- 配置验证
- 默认值设置
- 支持 Server、Worker、Queue、Database 配置

### 7. 数据模型 (internal/model/)

- Task 实体定义
- TaskEvent 事件记录
- TaskStatus 状态枚举
- TaskPriority 优先级枚举

### 8. Handler 层 (internal/handler/)

实现 gRPC 处理器：
- CreateTask - 创建任务
- GetTask - 获取任务
- ListTasks - 批量获取任务
- UpdateTask - 更新任务

### 9. Server 层 (internal/server/)

gRPC/HTTP 服务器：
- gRPC 服务端
- HTTP 网关
- 健康检查

### 10. Middleware 层 (internal/middleware/)

通用中间件：
- 日志中间件
- 错误处理中间件

## 📡 API 文档

### Simple RPC

```protobuf
service TaskService {
    rpc CreateTask(CreateTaskRequest) returns (Task);
    rpc GetTask(GetTaskRequest) returns (Task);
    rpc ListTasks(ListTasksRequest) returns (ListTasksResponse);
    rpc UpdateTask(UpdateTaskRequest) returns (Task);
}
```

### Request/Response 消息

**CreateTaskRequest:**
- name: string (required)
- description: string
- priority: TaskPriority
- task_type: string
- input_params: map<string, string>
- dependencies: repeated string
- max_retries: int32
- created_by: string

**GetTaskRequest:**
- id: string (required)
- include_events: bool

**ListTasksRequest:**
- page: int32
- page_size: int32
- status_filter: repeated TaskStatus
- keyword: string
- task_type: string
- priority: TaskPriority

**UpdateTaskRequest:**
- id: string (required)
- status: TaskStatus
- output_result: map<string, string>
- error_message: string
- retry_count: int32

## 📝 任务状态

| 状态 | 描述 |
|------|------|
| PENDING | 等待执行 |
| RUNNING | 执行中 |
| SUCCEEDED | 执行成功 |
| FAILED | 执行失败 |
| CANCELLED | 已取消 |
| TIMEOUT | 执行超时 |

## 📝 任务优先级

| 优先级 | 描述 |
|--------|------|
| LOW | 低优先级 |
| NORMAL | 普通优先级 |
| HIGH | 高优先级 |
| URGENT | 紧急优先级 |

## 🧪 测试

```bash
# 运行所有测试
go test ./... -v

# 覆盖率报告
go test ./... -cover

# 运行特定包测试
go test ./internal/service -v

# 运行特定测试
go test ./internal/service -v -run TestTaskService_CreateTask
```

### 测试覆盖

| 包 | 测试数 | 描述 |
|-----|--------|------|
| model | 9 | 数据模型单元测试 |
| repository | 12 | SQLite 持久化测试 |
| service | 12 | 业务逻辑与调度器测试 |

## 📄 许可证

MIT

---

## 📌 更新日志

### v0.2.0 (2026-02-19)
- ✅ Service 层实现
- ✅ 任务调度器 (Scheduler)
- ✅ 工作池 (WorkerPool)
- ✅ 状态机 (StateMachine)
- ✅ 依赖检查器 (DependencyChecker)
- ✅ 完整的单元测试与集成测试
- ✅ README 文档完善

### v0.1.0 (2026-02-14)
- ✅ 项目初始化
- ✅ 配置系统扩展 (WorkerConfig, QueueConfig, DatabaseConfig)
- ✅ 完整配置验证逻辑
- ✅ Task 数据模型
- ✅ SQLite 持久化层 (Repository)
- ✅ 错误处理模块
- ✅ Handler 层
- ✅ Server 层
- ✅ Middleware 层
