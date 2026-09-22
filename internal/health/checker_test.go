package health

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"liberation-guardian/internal/config"
	"liberation-guardian/pkg/types"
)

// fakeAIClient lets these tests drive the only input the checker actually has.
type fakeAIClient struct {
	healthy bool
	calls   int
}

func (f *fakeAIClient) SendRequest(context.Context, *types.AIRequest) (*types.AIResponse, error) {
	return nil, nil
}

func (f *fakeAIClient) IsHealthy(context.Context) bool {
	f.calls++
	return f.healthy
}

func newRouter(aiHealthy *bool) (*gin.Engine, *fakeAIClient) {
	gin.SetMode(gin.TestMode)
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	var client *fakeAIClient
	var checker *Checker
	if aiHealthy != nil {
		client = &fakeAIClient{healthy: *aiHealthy}
		checker = NewChecker(&config.Config{}, logger, client)
	} else {
		// nil interface: the checker skips the AI probe entirely
		checker = NewChecker(&config.Config{}, logger, nil)
	}

	r := gin.New()
	r.GET("/health", checker.HealthCheck)
	r.GET("/ready", checker.ReadinessCheck)
	return r, client
}

func get(t *testing.T, r *gin.Engine, path string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("%s did not return JSON: %v (%s)", path, err, w.Body.String())
	}
	return w, body
}

func TestHealthCheckHealthy(t *testing.T) {
	healthy := true
	r, client := newRouter(&healthy)

	w, body := get(t, r, "/health")
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	if body["status"] != "healthy" {
		t.Errorf("status field = %v, want healthy", body["status"])
	}
	if body["ai_client_healthy"] != true {
		t.Errorf("ai_client_healthy = %v, want true", body["ai_client_healthy"])
	}
	if client.calls != 1 {
		t.Errorf("AI client probed %d times, want 1", client.calls)
	}
}

// This is the behaviour worth knowing about: /health reports 503 when the AI
// provider is unreachable, and the Dockerfile's HEALTHCHECK polls /health with
// --retries=3 --interval=30s. So roughly 90s of Anthropic being unavailable
// marks the container unhealthy and the orchestrator replaces it — an outage in
// a third-party API becomes a restart loop in a service that is otherwise fine
// and still able to receive webhooks.
//
// Liveness should reflect "is this process working", not "is my upstream up".
// That distinction is what /ready is for, and it already reports it.
func TestHealthCheckReports503WhenAIProviderIsDown(t *testing.T) {
	healthy := false
	r, _ := newRouter(&healthy)

	w, body := get(t, r, "/health")
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503 (current behaviour)", w.Code)
	}
	if body["status"] != "degraded" {
		t.Errorf("status field = %v, want degraded", body["status"])
	}
	t.Log("NOTE: liveness is coupled to the AI provider — see comment above this test")
}

func TestHealthCheckWithoutAIClient(t *testing.T) {
	r, _ := newRouter(nil)

	w, body := get(t, r, "/health")
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 when no AI client is configured", w.Code)
	}
	if _, present := body["ai_client_healthy"]; present {
		t.Error("ai_client_healthy should be absent when there is no AI client")
	}
}

func TestReadinessReflectsAIClient(t *testing.T) {
	for _, healthy := range []bool{true, false} {
		h := healthy
		t.Run(map[bool]string{true: "ready", false: "not ready"}[h], func(t *testing.T) {
			r, _ := newRouter(&h)

			w, body := get(t, r, "/ready")

			wantCode := http.StatusOK
			if !h {
				wantCode = http.StatusServiceUnavailable
			}
			if w.Code != wantCode {
				t.Errorf("status = %d, want %d", w.Code, wantCode)
			}
			if body["ready"] != h {
				t.Errorf("ready = %v, want %v", body["ready"], h)
			}

			checks, ok := body["checks"].(map[string]any)
			if !ok {
				t.Fatalf("checks missing or wrong shape: %v", body["checks"])
			}
			if checks["ai_client"] != h {
				t.Errorf("checks.ai_client = %v, want %v", checks["ai_client"], h)
			}
		})
	}
}

// The previous suite accepted "200 or 503" from /ready, which passes whatever
// the service reports. Guard the one case that must never regress: a readiness
// probe that claims ready while its dependency check says otherwise.
func TestReadinessNeverClaimsReadyWhileACheckFails(t *testing.T) {
	healthy := false
	r, _ := newRouter(&healthy)

	w, body := get(t, r, "/ready")
	if body["ready"] == true && w.Code == http.StatusOK {
		t.Fatal("readiness reported ready while the AI client check was failing")
	}
}
