package handler

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"grpc-hello/api/dto"
)

// NewTooManyNamesError 创建过多名称错误
func NewTooManyNamesError(max int) error {
	return status.Error(
		codes.InvalidArgument,
		dto.ErrTooManyNames.Message,
	)
}

// NewStatsDisabledError 创建统计禁用错误
func NewStatsDisabledError() error {
	return status.Error(
		codes.FailedPrecondition,
		dto.ErrStatsDisabled.Message,
	)
}

// NewValidationError 创建验证错误（带详细信息）
func NewValidationError(message string, details string) error {
	st := status.New(codes.InvalidArgument, message)
	// 添加错误详情
	return st.Err()
}

// NewBadRequestError 创建400错误
func NewBadRequestError(message string) error {
	return status.Error(codes.InvalidArgument, message)
}

// NewNotFoundError 创建404错误
func NewNotFoundError(message string) error {
	return status.Error(codes.NotFound, message)
}

// NewInternalError 创建500错误
func NewInternalError(message string) error {
	return status.Error(codes.Internal, message)
}

// ErrorToGRPCStatus 将自定义错误转换为gRPC状态
func ErrorToGRPCStatus(err error) error {
	if err == nil {
		return nil
	}

	// 如果已经是gRPC状态错误，直接返回
	if _, ok := status.FromError(err); ok {
		return err
	}

	// 转换为内部错误
	return status.Error(codes.Internal, err.Error())
}

// ExtractErrorCode 从错误中提取业务错误码
func ExtractErrorCode(err error) dto.ErrorCode {
	if err == nil {
		return dto.CodeSuccess
	}

	// 尝试从gRPC状态错误中提取
	if st, ok := status.FromError(err); ok {
		switch st.Code() {
		case codes.InvalidArgument:
			return dto.CodeBadRequest
		case codes.NotFound:
			return dto.CodeNotFound
		case codes.AlreadyExists:
			return dto.CodeTooManyNames
		case codes.FailedPrecondition:
			return dto.CodeStatsDisabled
		case codes.Internal:
			return dto.CodeInternalError
		}
	}

	// 尝试从自定义错误中提取
	if be, ok := err.(*dto.Error); ok {
		return be.Code
	}

	return dto.CodeInternalError
}
