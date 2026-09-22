package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"liberation-guardian/internal/config"
	"liberation-guardian/pkg/types"
)

// The suite in tests/ asserted only that handlers did not return 5xx, which a
// 404 and a 400 both satisfy, and it never read the event channel — so the
// whole receive-to-queue path could drop every event and stay green. These
// tests assert exact status codes and read the channel.

const sentryIssuePayload = `{
  "action": "created",
  "data": {"issue": {
    "id": "1234",
    "title": "TypeError: undefined is not a function",
    "level": "error",
    "platform": "javascript",
    "message": "boom",
    "firstSeen": "2026-01-02T03:04:05Z",
    "count": 7,
    "permalink": "https://sentry.io/issues/1234",
    "project": {"name": "liberation-web", "slug": "prod"}
  }}
}`

const githubRunPayload = `{
  "action": "completed",
  "workflow_run": {"name": "CI", "conclusion": "failure"},
  "repository": {"name": "liberation-platform", "full_name": "org/liberation-platform"}
}`

func testConfig() *config.Config {
	cfg := &config.Config{}
	cfg.Core.Environment = "test"
	cfg.Integrations.Observability.Sentry.Enabled = true
	cfg.Integrations.Observability.Prometheus.Enabled = true
	cfg.Integrations.Observability.Grafana.Enabled = true
	cfg.Integrations.SourceControl.GitHub.Enabled = true
	return cfg
}

// newTestReceiver returns a receiver plus the channel its events land on.
func newTestReceiver(t *testing.T, cfg *config.Config, buffer int) (*gin.Engine, chan *types.LiberationGuardianEvent) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	eventChan := make(chan *types.LiberationGuardianEvent, buffer)
	receiver := NewReceiver(cfg, logger, eventChan)

	router := gin.New()
	router.Use(gin.Recovery())
	receiver.SetupRoutes(router)

	return router, eventChan
}

func post(t *testing.T, router *gin.Engine, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// mustReceive fails if no event was queued. This is the assertion the previous
// suite lacked entirely.
func mustReceive(t *testing.T, ch chan *types.LiberationGuardianEvent) *types.LiberationGuardianEvent {
	t.Helper()
	select {
	case ev := <-ch:
		if ev == nil {
			t.Fatal("received a nil event")
		}
		return ev
	default:
		t.Fatal("no event was queued — the webhook was accepted but nothing reached the pipeline")
		return nil
	}
}

func mustNotReceive(t *testing.T, ch chan *types.LiberationGuardianEvent) {
	t.Helper()
	select {
	case ev := <-ch:
		t.Fatalf("an event was queued when none should have been: %+v", ev)
	default:
	}
}

func TestSentryWebhookDeliversEvent(t *testing.T) {
	router, events := newTestReceiver(t, testConfig(), 4)

	w := post(t, router, "/webhook/sentry", sentryIssuePayload, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}

	ev := mustReceive(t, events)

	if ev.Source != string(types.SourceSentry) {
		t.Errorf("Source = %q, want %q", ev.Source, types.SourceSentry)
	}
	if ev.Type != "created" {
		t.Errorf("Type = %q, want %q (the Sentry action)", ev.Type, "created")
	}
	if ev.Title != "TypeError: undefined is not a function" {
		t.Errorf("Title = %q", ev.Title)
	}
	if ev.Severity != types.SeverityCritical {
		t.Errorf("Severity = %q, want critical for level=error", ev.Severity)
	}
	if ev.Service != "liberation-web" {
		t.Errorf("Service = %q, want the project name", ev.Service)
	}
	if ev.Environment != "prod" {
		t.Errorf("Environment = %q, want the project slug", ev.Environment)
	}
	if ev.Fingerprint == "" {
		t.Error("Fingerprint is empty — deduplication cannot work")
	}
	if got := ev.Metadata["sentry_issue_id"]; got != "1234" {
		t.Errorf("metadata sentry_issue_id = %v, want 1234", got)
	}

	// The response should hand back the same event id that was queued.
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not JSON: %v", err)
	}
	if body["event_id"] != ev.ID {
		t.Errorf("response event_id = %v but queued event ID = %v", body["event_id"], ev.ID)
	}
}

func TestGitHubWebhookDeliversEvent(t *testing.T) {
	router, events := newTestReceiver(t, testConfig(), 4)

	w := post(t, router, "/webhook/github", githubRunPayload,
		map[string]string{"X-GitHub-Event": "workflow_run"})
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}

	ev := mustReceive(t, events)
	if ev.Source != string(types.SourceGitHub) {
		t.Errorf("Source = %q, want github", ev.Source)
	}
	if ev.Type != "workflow_run" {
		t.Errorf("Type = %q, want workflow_run (from the X-GitHub-Event header)", ev.Type)
	}
}

// Severity mapping drives triage and escalation, so pin it.
func TestSentrySeverityMapping(t *testing.T) {
	p := NewSentryProcessor(logrus.New())
	cases := map[string]types.Severity{
		"fatal":   types.SeverityCritical,
		"error":   types.SeverityCritical,
		"ERROR":   types.SeverityCritical, // mapping lowercases first
		"warning": types.SeverityHigh,
		"info":    types.SeverityMedium,
		"debug":   types.SeverityLow,
		"":        types.SeverityLow,
	}
	for level, want := range cases {
		if got := p.mapSentrySeverity(level); got != want {
			t.Errorf("mapSentrySeverity(%q) = %q, want %q", level, got, want)
		}
	}
}

// Same issue must fingerprint identically, different issues must not —
// otherwise deduplication either misses duplicates or collapses distinct ones.
func TestSentryFingerprintStability(t *testing.T) {
	p := NewSentryProcessor(logrus.New())

	a := p.generateSentryFingerprint("1234", "boom")
	b := p.generateSentryFingerprint("1234", "boom")
	c := p.generateSentryFingerprint("9999", "boom")

	if a != b {
		t.Errorf("same issue produced different fingerprints: %q vs %q", a, b)
	}
	if a == c {
		t.Errorf("different issues collided on fingerprint %q", a)
	}
	if len(a) != 16 {
		t.Errorf("fingerprint length = %d, want 16", len(a))
	}
}

func TestUnsupportedSourceIsRejected(t *testing.T) {
	router, events := newTestReceiver(t, testConfig(), 4)

	// /webhook/gitlab is routed in SetupRoutes, but registerProcessors never
	// registers a GitLab processor, so the route exists and can never succeed.
	w := post(t, router, "/webhook/gitlab", `{"object_kind":"pipeline"}`,
		map[string]string{"X-Gitlab-Event": "Pipeline Hook"})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 for a source with no registered processor", w.Code)
	}
	mustNotReceive(t, events)
}

func TestDisabledIntegrationIsRejected(t *testing.T) {
	cfg := testConfig()
	cfg.Integrations.Observability.Sentry.Enabled = false
	router, events := newTestReceiver(t, cfg, 4)

	w := post(t, router, "/webhook/sentry", sentryIssuePayload, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 when the Sentry integration is disabled", w.Code)
	}
	mustNotReceive(t, events)
}

func TestMalformedJSONIsRejected(t *testing.T) {
	router, events := newTestReceiver(t, testConfig(), 4)

	w := post(t, router, "/webhook/sentry", `{"action":`, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 for malformed JSON", w.Code)
	}
	mustNotReceive(t, events)
}

// A full channel must not be reported as a successful receipt.
func TestFullChannelReturns503AndDropsEvent(t *testing.T) {
	router, events := newTestReceiver(t, testConfig(), 1)

	if w := post(t, router, "/webhook/sentry", sentryIssuePayload, nil); w.Code != http.StatusOK {
		t.Fatalf("first request status = %d, want 200", w.Code)
	}
	// Channel now holds 1 of 1.
	w := post(t, router, "/webhook/sentry", sentryIssuePayload, nil)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503 when the event channel is full", w.Code)
	}
	if len(events) != 1 {
		t.Errorf("channel length = %d, want 1 (the second event should be dropped, not buffered)", len(events))
	}
}

func TestUniversalWebhookDetectsSource(t *testing.T) {
	cases := []struct {
		name    string
		headers map[string]string
		body    string
		want    types.EventSource
	}{
		{"user agent", map[string]string{"User-Agent": "Sentry/1.0"}, sentryIssuePayload, types.SourceSentry},
		{"sentry hook header", map[string]string{"X-Sentry-Hook-Resource": "issue"}, sentryIssuePayload, types.SourceSentry},
		{"github event header", map[string]string{"X-GitHub-Event": "workflow_run"}, githubRunPayload, types.SourceGitHub},
		{"payload shape", nil, githubRunPayload, types.SourceGitHub}, // detected via "repository"
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			router, events := newTestReceiver(t, testConfig(), 4)

			w := post(t, router, "/webhook/", tc.body, tc.headers)
			if w.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
			}
			// The old test asserted only "not 5xx" and never checked which
			// source was detected, despite being named for it.
			ev := mustReceive(t, events)
			if ev.Source != string(tc.want) {
				t.Errorf("detected source = %q, want %q", ev.Source, tc.want)
			}
		})
	}
}

func TestUniversalWebhookRejectsUndetectableSource(t *testing.T) {
	router, events := newTestReceiver(t, testConfig(), 4)

	w := post(t, router, "/webhook/", `{"something":"unrecognised"}`, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 when the source cannot be detected", w.Code)
	}
	mustNotReceive(t, events)
}

func sign(secret string, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func TestSignatureValidation(t *testing.T) {
	const secret = "s3cret"
	payload := []byte(sentryIssuePayload)

	t.Run("valid signature accepted", func(t *testing.T) {
		if !ValidateHMAC(payload, sign(secret, payload), secret) {
			t.Error("a correctly signed payload was rejected")
		}
	})
	t.Run("wrong secret rejected", func(t *testing.T) {
		if ValidateHMAC(payload, sign("other", payload), secret) {
			t.Error("a payload signed with the wrong secret was accepted")
		}
	})
	t.Run("tampered payload rejected", func(t *testing.T) {
		sig := sign(secret, payload)
		if ValidateHMAC([]byte(`{"action":"tampered"}`), sig, secret) {
			t.Error("a tampered payload kept a valid signature")
		}
	})
	t.Run("non-hex signature rejected", func(t *testing.T) {
		if ValidateHMAC(payload, "sha256=not-hex", secret) {
			t.Error("a malformed signature was accepted")
		}
	})
	t.Run("empty signature rejected", func(t *testing.T) {
		if ValidateHMAC(payload, "", secret) {
			t.Error("an empty signature was accepted")
		}
	})
}

// Pins the current posture: with no secret env configured, GetWebhookSecret
// returns "" and validateWebhookSignature short-circuits to true, so webhooks
// are accepted unauthenticated. That is a deliberate default, not a bug, but it
// should fail loudly here if someone believes signatures are being enforced.
func TestUnsignedWebhookAcceptedWhenNoSecretConfigured(t *testing.T) {
	cfg := testConfig()
	if got := cfg.GetWebhookSecret("sentry"); got != "" {
		t.Fatalf("precondition: expected no configured secret, got %q", got)
	}

	router, events := newTestReceiver(t, cfg, 4)
	w := post(t, router, "/webhook/sentry", sentryIssuePayload, nil)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	mustReceive(t, events)
}

// Custom webhooks copy every request header into event metadata, which puts
// Authorization values into whatever stores or logs events downstream.
func TestCustomWebhookCopiesHeadersIntoMetadata(t *testing.T) {
	router, events := newTestReceiver(t, testConfig(), 4)

	w := post(t, router, "/webhook/custom/mytool", `{"hello":"world"}`,
		map[string]string{"Authorization": "Bearer super-secret-token"})
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	ev := mustReceive(t, events)
	if ev.Source != "mytool" {
		t.Errorf("Source = %q, want the :source path param", ev.Source)
	}

	got, ok := ev.Metadata["header_authorization"]
	if !ok {
		t.Skip("Authorization is no longer copied into metadata — the leak described here is fixed")
	}
	if got != "Bearer super-secret-token" {
		t.Fatalf("unexpected metadata value %v", got)
	}
	t.Log("NOTE: the Authorization header is copied verbatim into event metadata " +
		"and travels with the event into Redis and any downstream sink")
}
