package ai

import "testing"
import "github.com/everlier/claune/internal/config"

func TestAnalyze(t *testing.T) {
	c := config.ClauneConfig{}
	res, _ := AnalyzeToolIntent("bash", "rm -rf /", c)
	if res != "tool:start" {
		t.Error("Expected tool:start fallback when disabled")
	}
}
