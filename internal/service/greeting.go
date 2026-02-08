package service

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// Language 问候语言配置
type Language struct {
	Code     string
	Name     string
	Greeting string
}

// GreetingService 问候服务
type GreetingService struct {
	mu          sync.RWMutex
	totalReq    int64
	uniqueNames map[string]int64
	nameFreq    map[string]int64
	lastReq     time.Time
	maxGreetings int
	// 语言缓存
	langCodeMap    map[string]*Language
	langNameMap    map[string]*Language
	langAliasesMap map[string]string
}

// NewGreetingService 创建问候服务
func NewGreetingService(maxGreetings int) *GreetingService {
	svc := &GreetingService{
		uniqueNames: make(map[string]int64),
		nameFreq:    make(map[string]int64),
		maxGreetings: maxGreetings,
		langCodeMap:  make(map[string]*Language),
		langNameMap:  make(map[string]*Language),
		langAliasesMap: map[string]string{
			"chinese":  "zh",
			"spanish":  "es",
			"french":   "fr",
			"japanese": "ja",
			"korean":   "ko",
			"russian":  "ru",
			"german":   "de",
			"italian": "it",
		},
	}

	// 初始化语言缓存
	for i := range SupportedLanguages {
		lang := &SupportedLanguages[i]
		svc.langCodeMap[strings.ToLower(lang.Code)] = lang
		svc.langNameMap[strings.ToLower(lang.Name)] = lang
	}

	return svc
}

// SupportedLanguages 支持的语言列表
var SupportedLanguages = []Language{
	{Code: "en", Name: "English", Greeting: "Hello"},
	{Code: "zh", Name: "Chinese", Greeting: "你好"},
	{Code: "es", Name: "Spanish", Greeting: "Hola"},
	{Code: "fr", Name: "French", Greeting: "Bonjour"},
	{Code: "ja", Name: "Japanese", Greeting: "こんにちは"},
	{Code: "ko", Name: "Korean", Greeting: "안녕하세요"},
	{Code: "ru", Name: "Russian", Greeting: "Привет"},
	{Code: "de", Name: "German", Greeting: "Hallo"},
	{Code: "it", Name: "Italian", Greeting: "Ciao"},
}

// GetGreeting 获取问候语（使用缓存优化）
func (s *GreetingService) GetGreeting(language string) string {
	if language == "" {
		return SupportedLanguages[0].Greeting
	}

	lang := strings.ToLower(language)

	// 优先检查代码
	if langData, ok := s.langCodeMap[lang]; ok {
		return langData.Greeting
	}

	// 检查名称
	if langData, ok := s.langNameMap[lang]; ok {
		return langData.Greeting
	}

	// 检查别名
	if alias, ok := s.langAliasesMap[lang]; ok {
		return s.GetGreeting(alias)
	}

	return SupportedLanguages[0].Greeting
}

// BuildMessage 构建问候消息
func (s *GreetingService) BuildMessage(name, language, extraMsg string) string {
	greeting := s.GetGreeting(language)
	if extraMsg != "" {
		return fmt.Sprintf("%s %s! %s", greeting, name, extraMsg)
	}
	return fmt.Sprintf("%s %s!", greeting, name)
}

// UpdateStats 更新统计
func (s *GreetingService) UpdateStats(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.totalReq++
	if name != "" {
		lowerName := strings.ToLower(name)
		s.uniqueNames[lowerName]++
		s.nameFreq[lowerName]++
	}
	s.lastReq = time.Now()
}

// GetStats 获取统计信息（优化版 - 预分配map大小）
func (s *GreetingService) GetStats(nameFilter string, limit int) (totalReq, uniqueNames int32, nameFreq map[string]int32, lastReq int64) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	totalReq = int32(s.totalReq)
	uniqueNames = int32(len(s.uniqueNames))
	lastReq = s.lastReq.Unix()

	// 预计算过滤后的结果数量，避免重复分配
	filteredCount := 0
	if nameFilter == "" {
		filteredCount = len(s.nameFreq)
	} else {
		for name := range s.nameFreq {
			if strings.Contains(name, nameFilter) {
				filteredCount++
			}
		}
	}

	// 预分配map大小，减少内存重新分配
	resultSize := filteredCount
	if limit > 0 && limit < resultSize {
		resultSize = limit
	}
	result := make(map[string]int32, resultSize)

	// 如果不需要排序，直接返回过滤后的结果
	if nameFilter == "" && (limit <= 0 || limit >= len(s.nameFreq)) {
		for name, count := range s.nameFreq {
			result[name] = int32(count)
		}
		return totalReq, uniqueNames, result, lastReq
	}

	// 需要排序的情况
	type kv struct {
		Key   string
		Value int32
	}
	// 预分配切片大小
	sorted := make([]kv, 0, filteredCount)
	for name, count := range s.nameFreq {
		if nameFilter == "" || strings.Contains(name, nameFilter) {
			sorted = append(sorted, kv{name, int32(count)})
		}
	}

	// 快速排序（对于小数据集使用插入排序更高效）
	const maxInsertionSortSize = 64
	if len(sorted) <= maxInsertionSortSize {
		for i := 1; i < len(sorted); i++ {
			key := sorted[i]
			j := i - 1
			for j >= 0 && sorted[j].Value < key.Value {
				sorted[j+1] = sorted[j]
				j--
			}
			sorted[j+1] = key
		}
	} else {
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].Value > sorted[j].Value
		})
	}

	// 限制结果数量
	if len(sorted) > limit && limit > 0 {
		sorted = sorted[:limit]
	}

	// 填充结果map
	for _, kv := range sorted {
		result[kv.Key] = kv.Value
	}

	return totalReq, uniqueNames, result, lastReq
}

// GetMaxGreetings 获取最大问候数量
func (s *GreetingService) GetMaxGreetings() int {
	return s.maxGreetings
}
