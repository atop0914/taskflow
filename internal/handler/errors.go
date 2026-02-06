package handler

import (
	"google.golang.org/grpc/status"
	grpccodes "google.golang.org/grpc/codes"

	"grpc-hello/api/dto"
)

// NewTooManyNamesError 创建过多名称错误
func NewTooManyNamesError(max int) error {
	return status.Error(
		grpccodes.InvalidArgument,
		dto.ErrTooManyNames.Message,
	)
}

// NewStatsDisabledError 创建统计禁用错误
func NewStatsDisabledError() error {
	return status.Error(
		grpccodes.FailedPrecondition,
		dto.ErrStatsDisabled.Message,
	)
}
