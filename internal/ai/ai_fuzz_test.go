package ai

import (
	"testing"
	"github.com/everlier/claune/internal/config"
)

func FuzzDoAIRequest(f *testing.F) {
	f.Add(string([]byte{0x7f})) // Invalid URL control character
	f.Add("http://127.0.0.1:0")
	f.Add("invalid_url")

	f.Fuzz(func(t *testing.T, apiUrl string) {
		c := config.ClauneConfig{
			AI: config.AIConfig{
				Enabled: true,
				APIKey:  "dummy",
				APIURL:  apiUrl,
			},
		}
		
		reqBody := ClaudeRequest{
			Model: "dummy",
			MaxTokens: 10,
		}
		
		// This should return an error, but NOT panic
		doAIRequest(c, reqBody)
	})
}
