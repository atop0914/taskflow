package service

import (
	"testing"
)

func TestGreetingService_GetGreeting(t *testing.T) {
	s := NewGreetingService(100)

	tests := []struct {
		name     string
		language string
		want     string
	}{
		{"empty language", "", "Hello"},
		{"english", "en", "Hello"},
		{"chinese", "zh", "你好"},
		{"spanish", "es", "Hola"},
		{"french", "fr", "Bonjour"},
		{"japanese", "ja", "こんにちは"},
		{"korean", "ko", "안녕하세요"},
		{"russian", "ru", "Привет"},
		{"german", "de", "Hallo"},
		{"italian", "it", "Ciao"},
		{"chinese alias", "chinese", "你好"},
		{"unknown language", "xyz", "Hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := s.GetGreeting(tt.language); got != tt.want {
				t.Errorf("GetGreeting(%q) = %q, want %q", tt.language, got, tt.want)
			}
		})
	}
}

func TestGreetingService_BuildMessage(t *testing.T) {
	s := NewGreetingService(100)

	tests := []struct {
		name     string
		language string
		extraMsg string
		want     string
	}{
		{"basic message", "en", "", "Hello World!"},
		{"with extra message", "zh", "欢迎", "你好 World! 欢迎"},
		{"default language", "", "", "Hello World!"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := s.BuildMessage("World", tt.language, tt.extraMsg); got != tt.want {
				t.Errorf("BuildMessage() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGreetingService_UpdateStats(t *testing.T) {
	s := NewGreetingService(100)

	// Initial stats should be zero
	total, unique, _, _ := s.GetStats("", 10)
	if total != 0 || unique != 0 {
		t.Errorf("Initial stats should be zero, got total=%d, unique=%d", total, unique)
	}

	// Update stats
	s.UpdateStats("Alice")
	s.UpdateStats("Bob")
	s.UpdateStats("alice") // Case insensitive

	total, unique, freq, _ := s.GetStats("", 10)

	if total != 3 {
		t.Errorf("Total requests = %d, want 3", total)
	}
	if unique != 2 {
		t.Errorf("Unique names = %d, want 2", unique)
	}
	if freq["alice"] != 2 {
		t.Errorf("Alice frequency = %d, want 2", freq["alice"])
	}
}
