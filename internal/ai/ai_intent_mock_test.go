package ai

import (
	"testing"

	"github.com/everlier/claune/internal/config"
)

func TestAnalyzeToolIntent_DisabledAI(t *testing.T) {
	_ = setupHermeticAITest(t)
	intent, err := AnalyzeToolIntent("tool:success", "bash", "echo hello", config.ClauneConfig{AI: config.AIConfig{Enabled: false}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if intent != "tool:success" {
		t.Fatalf("intent = %q, want %q", intent, "tool:success")
	}
}

func TestAnalyzeToolIntent_MissingKeyDoesNotConsumeHostEnv(t *testing.T) {
	_ = setupHermeticAITest(t)
	t.Setenv("ANTHROPIC_API_KEY", "")
	_, err := AnalyzeToolIntent("tool:success", "bash", "echo hello", config.ClauneConfig{AI: config.AIConfig{Enabled: true}})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "AI enabled but no ANTHROPIC_API_KEY found" {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestDiagnoseInstallFailure_DisabledAI(t *testing.T) {
	_ = setupHermeticAITest(t)
	diagnosis := DiagnoseInstallFailure(nil, config.ClauneConfig{AI: config.AIConfig{Enabled: false}})
	if diagnosis != "AI diagnostics disabled. Please check your Claude config and permissions manually." {
		t.Fatalf("diagnosis = %q", diagnosis)
	}
}

func TestAnalyzeResponseSentiment_DisabledAI(t *testing.T) {
	_ = setupHermeticAITest(t)
	sentiment, strategy, err := AnalyzeResponseSentiment("Great job!", config.ClauneConfig{AI: config.AIConfig{Enabled: false}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sentiment != "" || strategy != "" {
		t.Fatalf("got (%q, %q), want empty values", sentiment, strategy)
	}
}

func TestAnalyzeResponseSentiment_EmptyResponseSkipsAI(t *testing.T) {
	_ = setupHermeticAITest(t)
	sentiment, strategy, err := AnalyzeResponseSentiment("   ", config.ClauneConfig{AI: config.AIConfig{Enabled: true}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sentiment != "" || strategy != "" {
		t.Fatalf("got (%q, %q), want empty values", sentiment, strategy)
	}
}
