package ai

import (
	"testing"
	"github.com/everlier/claune/internal/config"
)

func TestAnalyzeToolIntent_DisabledAI(t *testing.T) {
	c := config.ClauneConfig{
		AI: config.AIConfig{
			Enabled: false,
		},
	}
	intent, err := AnalyzeToolIntent("tool:success", "bash", "echo hello", c)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if intent != "tool:success" {
		t.Errorf("Expected tool:success, got %s", intent)
	}
}

func TestDiagnoseInstallFailure_DisabledAI(t *testing.T) {
	c := config.ClauneConfig{
		AI: config.AIConfig{
			Enabled: false,
		},
	}
	diagnosis := DiagnoseInstallFailure(nil, c)
	if diagnosis != "AI diagnostics disabled. Please check your Claude config and permissions manually." {
		t.Errorf("Unexpected diagnosis: %s", diagnosis)
	}
}

func TestAnalyzeResponseSentiment_DisabledAI(t *testing.T) {
	c := config.ClauneConfig{
		AI: config.AIConfig{
			Enabled: false,
		},
	}
	sentiment, strategy, err := AnalyzeResponseSentiment("Great job!", c)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if sentiment != "" || strategy != "" {
		t.Errorf("Expected empty strings, got %s and %s", sentiment, strategy)
	}
}
