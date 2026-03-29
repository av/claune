package config

import (
	"encoding/json"
	"testing"
)

func TestShouldMute(t *testing.T) {
	c := ClauneConfig{}
	_ = c.ShouldMute()
}

func TestParseNewConfig(t *testing.T) {
	data := []byte(`{
		"sounds": {
			"success": {
				"paths": ["test1.mp3", "test2.mp3"],
				"strategy": "round_robin"
			}
		}
	}`)
	var c ClauneConfig
	err := json.Unmarshal(data, &c)
	if err != nil {
		t.Fatalf("Failed to parse new config: %v", err)
	}
	sc, ok := c.Sounds["success"]
	if !ok {
		t.Fatalf("Expected success config")
	}
	if len(sc.Paths) != 2 || sc.Paths[0] != "test1.mp3" || sc.Strategy != "round_robin" {
		t.Errorf("Parsed incorrectly: %+v", sc)
	}
}

func TestParseOldConfigFailsGracefully(t *testing.T) {
	data := []byte(`{
		"sounds": {
			"success": "test1.mp3"
		}
	}`)
	var c ClauneConfig
	err := json.Unmarshal(data, &c)
	if err == nil {
		t.Fatalf("Expected clear parsing failure for old config shape, but got no error")
	}
}
