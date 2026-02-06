package handler

import (
	"context"
	"log"
	"time"

	helloworldpb "grpc-hello/proto/helloworld"
	"grpc-hello/internal/service"
)

// GreeterHandler gRPC问候处理器
type GreeterHandler struct {
	helloworldpb.UnimplementedGreeterServer
	greetingService *service.GreetingService
}

// NewGreeterHandler 创建gRPC处理器
func NewGreeterHandler(greetingService *service.GreetingService) *GreeterHandler {
	return &GreeterHandler{
		greetingService: greetingService,
	}
}

// SayHello 实现问候接口
func (h *GreeterHandler) SayHello(ctx context.Context, req *helloworldpb.HelloRequest) (*helloworldpb.HelloReply, error) {
	name := req.GetNameTest()
	if name == "" {
		name = "World"
	}

	// 更新统计
	h.greetingService.UpdateStats(name)

	// 构建消息
	message := h.greetingService.BuildMessage(name, req.GetLanguage(), "")

	reply := &helloworldpb.HelloReply{
		TestMessage: message,
		Timestamp:   time.Now().Unix(),
		Language:    req.GetLanguage(),
		Tags:        req.GetTags(),
	}

	log.Printf("[gRPC] SayHello: %s (lang: %s)", name, req.GetLanguage())

	return reply, nil
}

// SayHelloMultiple 实现批量问候接口
func (h *GreeterHandler) SayHelloMultiple(ctx context.Context, req *helloworldpb.HelloMultipleRequest) (*helloworldpb.HelloMultipleReply, error) {
	maxGreetings := h.greetingService.GetMaxGreetings()
	if len(req.Names) > maxGreetings {
		log.Printf("[gRPC] SayHelloMultiple: too many names (%d > %d)", len(req.Names), maxGreetings)
		return nil, NewTooManyNamesError(maxGreetings)
	}

	var greetings []*helloworldpb.HelloReply
	for _, name := range req.Names {
		h.greetingService.UpdateStats(name)
		message := h.greetingService.BuildMessage(name, "", req.GetCommonMessage())

		greetings = append(greetings, &helloworldpb.HelloReply{
			TestMessage: message,
			Timestamp:   time.Now().Unix(),
		})
	}

	log.Printf("[gRPC] SayHelloMultiple: %d greetings", len(req.Names))

	return &helloworldpb.HelloMultipleReply{
		Greetings:   greetings,
		TotalCount:  int32(len(greetings)),
	}, nil
}

// GetGreetingStats 获取统计信息
func (h *GreeterHandler) GetGreetingStats(ctx context.Context, req *helloworldpb.GreetingStatsRequest) (*helloworldpb.GreetingStatsReply, error) {
	totalReq, uniqueNames, nameFreq, lastReq := h.greetingService.GetStats(req.GetNameFilter(), 10)

	log.Printf("[gRPC] GetGreetingStats: total=%d, unique=%d", totalReq, uniqueNames)

	return &helloworldpb.GreetingStatsReply{
		TotalRequests:   totalReq,
		UniqueNames:     uniqueNames,
		NameFrequency:   nameFreq,
		LastRequestTime: lastReq,
	}, nil
}
