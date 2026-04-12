package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/everlier/claune/internal/config"
)

type hermeticAITestEnv struct {
	root       string
	home       string
	configHome string
	cacheHome  string
	dataHome   string
	stateHome  string

	sleeps []time.Duration
}

func setupHermeticAITest(t *testing.T) *hermeticAITestEnv {
	t.Helper()

	root := t.TempDir()
	home := filepath.Join(root, "home")
	base := filepath.Join(root, "xdg")
	configHome := filepath.Join(base, "config")
	cacheHome := filepath.Join(base, "cache")
	dataHome := filepath.Join(base, "data")
	stateHome := filepath.Join(base, "state")

	for _, dir := range []string{home, configHome, cacheHome, dataHome, stateHome} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}

	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("XDG_CACHE_HOME", cacheHome)
	t.Setenv("XDG_DATA_HOME", dataHome)
	t.Setenv("XDG_STATE_HOME", stateHome)
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("HTTP_PROXY", "")
	t.Setenv("HTTPS_PROXY", "")
	t.Setenv("ALL_PROXY", "")
	t.Setenv("NO_PROXY", "")
	t.Setenv("http_proxy", "")
	t.Setenv("https_proxy", "")
	t.Setenv("all_proxy", "")
	t.Setenv("no_proxy", "")

	env := &hermeticAITestEnv{
		root:       root,
		home:       home,
		configHome: configHome,
		cacheHome:  cacheHome,
		dataHome:   dataHome,
		stateHome:  stateHome,
	}

	prevClientFactory := newAIHTTPClient
	newAIHTTPClient = func() *http.Client {
		transport := &http.Transport{
			Proxy: nil,
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				host, _, err := net.SplitHostPort(addr)
				if err != nil {
					return nil, err
				}
				if host != "127.0.0.1" && host != "::1" && host != "localhost" {
					return nil, fmt.Errorf("non-local outbound HTTP blocked in tests: %s", addr)
				}
				return (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, network, addr)
			},
		}
		return &http.Client{Timeout: 5 * time.Second, Transport: transport}
	}
	t.Cleanup(func() {
		newAIHTTPClient = prevClientFactory
	})

	prevSleep := aiSleep
	aiSleep = func(d time.Duration) {
		env.sleeps = append(env.sleeps, d)
	}
	t.Cleanup(func() {
		aiSleep = prevSleep
	})

	configPath := config.ConfigFilePath()
	if !strings.HasPrefix(configPath, filepath.Join(configHome, "claune")+string(filepath.Separator)) {
		t.Fatalf("config path %q not rooted in test XDG config %q", configPath, configHome)
	}

	return env
}

func (e *hermeticAITestEnv) configPath() string {
	return config.ConfigFilePath()
}

func (e *hermeticAITestEnv) legacyConfigPath() string {
	return filepath.Join(e.home, ".claune.json")
}

func (e *hermeticAITestEnv) assertNoHostSideEffects(t *testing.T) {
	t.Helper()
	for _, path := range []string{
		e.legacyConfigPath(),
		filepath.Join(e.cacheHome, "claune"),
		filepath.Join(e.stateHome, "claune", "state.json"),
	} {
		if _, err := os.Stat(path); err == nil {
			t.Fatalf("unexpected test side effect at %s", path)
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat %s: %v", path, err)
		}
	}

	configPath := e.configPath()
	if strings.Contains(configPath, "/.config/claune/config.json") && !strings.HasPrefix(configPath, e.root) {
		t.Fatalf("config path escaped temp root: %s", configPath)
	}
	if !strings.HasPrefix(configPath, e.root) {
		t.Fatalf("config path %q not under temp root %q", configPath, e.root)
	}
	if strings.Contains(configPath, "/home/") && !strings.HasPrefix(configPath, e.root) {
		t.Fatalf("config path unexpectedly points at real home: %s", configPath)
	}
}

func (e *hermeticAITestEnv) loadSavedConfig(t *testing.T) config.ClauneConfig {
	t.Helper()
	data, err := os.ReadFile(e.configPath())
	if err != nil {
		t.Fatalf("read saved config: %v", err)
	}
	var cfg config.ClauneConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal saved config: %v", err)
	}
	return cfg
}

type anthropicRequestCapture struct {
	Path        string
	Method      string
	APIKey      string
	Version     string
	ContentType string
	Request     ClaudeRequest
	RawBody     []byte
	Calls       int
}

func createMockAnthropicServer(t *testing.T, status int, headers map[string]string, handler func(r *http.Request, req ClaudeRequest, capture *anthropicRequestCapture) any) (*httptest.Server, *anthropicRequestCapture) {
	t.Helper()

	capture := &anthropicRequestCapture{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capture.Calls++
		capture.Path = r.URL.Path
		capture.Method = r.Method
		capture.APIKey = r.Header.Get("x-api-key")
		capture.Version = r.Header.Get("anthropic-version")
		capture.ContentType = r.Header.Get("content-type")

		if capture.Path != "/v1/messages" {
			t.Errorf("request path = %q, want %q", capture.Path, "/v1/messages")
		}
		if capture.Method != http.MethodPost {
			t.Errorf("request method = %q, want %q", capture.Method, http.MethodPost)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		capture.RawBody = body
		if len(body) > 0 {
			if err := json.Unmarshal(body, &capture.Request); err != nil {
				t.Fatalf("unmarshal request body: %v", err)
			}
		}

		for k, v := range headers {
			w.Header().Set(k, v)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if handler == nil {
			return
		}
		if err := json.NewEncoder(w).Encode(handler(r, capture.Request, capture)); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))

	return server, capture
}

func mockConfig(serverURL string) config.ClauneConfig {
	return config.ClauneConfig{
		AI: config.AIConfig{
			Enabled: true,
			APIKey:  "test-key",
			APIURL:  serverURL,
			Model:   "claude-3-opus-test",
		},
	}
}
