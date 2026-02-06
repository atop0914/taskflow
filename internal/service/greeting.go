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
}

// NewGreetingService 创建问候服务
func NewGreetingService(maxGreetings int) *GreetingService {
	return &GreetingService{
		uniqueNames: make(map[string]int64),
		nameFreq:    make(map[string]int64),
		maxGreetings: maxGreetings,
	}
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

// GetGreeting 获取问候语
func (s *GreetingService) GetGreeting(language string) string {
	if language == "" {
		return SupportedLanguages[0].Greeting
	}

	lang := strings.ToLower(language)
	for _, l := range SupportedLanguages {
		if strings.ToLower(l.Code) == lang || strings.ToLower(l.Name) == lang {
			return l.Greeting
		}
	}

	// Check aliases
	aliases := map[string]string{
		"chinese":  "zh",
		"spanish":  "es",
		"french":   "fr",
		"japanese": "ja",
		"korean":   "ko",
		"russian":  "ru",
		"german":   "de",
		"italian": "it",
	}

	if alias, ok := aliases[lang]; ok {
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

// GetStats 获取统计信息
func (s *GreetingService) GetStats(nameFilter string, limit int) (totalReq, uniqueNames int32, nameFreq map[string]int32, lastReq int64) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	totalReq = int32(s.totalReq)
	uniqueNames = int32(len(s.uniqueNames))
	lastReq = s.lastReq.Unix()

	// Filter and sort
	freq := make(map[string]int32)
	for name, count := range s.nameFreq {
		if nameFilter == "" || strings.Contains(name, nameFilter) {
			freq[name] = int32(count)
		}
	}

	// Sort by frequency
	type kv struct {
		Key   string
		Value int32
	}
	var sorted []kv
	for k, v := range freq {
		sorted = append(sorted, kv{k, v})
	}

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Value > sorted[j].Value
	})

	// Limit results
	if len(sorted) > limit {
		sorted = sorted[:limit]
	}

	result := make(map[string]int32)
	for _, kv := range sorted {
		result[kv.Key] = kv.Value
	}

	return totalReq, uniqueNames, result, lastReq
}

// GetMaxGreetings 获取最大问候数量
func (s *GreetingService) GetMaxGreetings() int {
	return s.maxGreetings
}
