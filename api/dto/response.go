package dto

import "time"

// BaseResponse 基础响应结构
type BaseResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	TraceID string      `json:"trace_id,omitempty"`
	Time    int64       `json:"time"`
}

// NewSuccessResponse 创建成功响应
func NewSuccessResponse(data interface{}) *BaseResponse {
	return &BaseResponse{
		Code:    0,
		Message: "success",
		Data:    data,
		Time:    time.Now().Unix(),
	}
}

// NewErrorResponse 创建错误响应
func NewErrorResponse(code int, message string) *BaseResponse {
	return &BaseResponse{
		Code:    code,
		Message: message,
		Time:    time.Now().Unix(),
	}
}

// HelloRequest 问候请求
type HelloRequest struct {
	Name     string   `json:"name" binding:"required"`
	Language string   `json:"language"`
	Tags     []string `json:"tags"`
}

// HelloResponse 问候响应
type HelloResponse struct {
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
	Language  string `json:"language"`
}

// HelloMultipleRequest 批量问候请求
type HelloMultipleRequest struct {
	Names         []string `json:"names" binding:"required,min=1,max=100"`
	CommonMessage string   `json:"common_message"`
	Language      string   `json:"language"`
}

// HelloMultipleResponse 批量问候响应
type HelloMultipleResponse struct {
	Greetings  []*HelloResponse `json:"greetings"`
	TotalCount int32             `json:"total_count"`
}

// StatsResponse 统计响应
type StatsResponse struct {
	TotalRequests   int32            `json:"total_requests"`
	UniqueNames     int32            `json:"unique_names"`
	NameFrequency   map[string]int32 `json:"name_frequency"`
	LastRequestTime int64            `json:"last_request_time"`
}

// HealthResponse 健康检查响应
type HealthResponse struct {
	Status    string `json:"status"`
	GRPCPort  string `json:"grpc_port"`
	HTTPPort  string `json:"http_port"`
	Uptime    int64  `json:"uptime"`
	Timestamp int64  `json:"timestamp"`
}
