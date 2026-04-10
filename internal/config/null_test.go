package config

import (
	"encoding/json"
	"testing"
)

func TestNullStruct(t *testing.T) {
	data := []byte(`{"sounds": {"alert": null}}`)
	var c ClauneConfig
	err := json.Unmarshal(data, &c)
	t.Logf("err: %v", err)
	t.Logf("config: %+v", c)
}
