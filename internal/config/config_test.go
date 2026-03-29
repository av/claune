package config

import "testing"

func TestShouldMute(t *testing.T) {
	c := ClauneConfig{}
	_ = c.ShouldMute()
}
