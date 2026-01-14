package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"liberation-guardian/internal/config"
	"liberation-guardian/pkg/types"
)

// Receiver handles incoming webhooks from various observability sources
type Receiver struct {
	config     *config.Config
	logger     *logrus.Logger
	eventChan  chan *types.LiberationGuardianEvent
	processors map[types.EventSource]Processor
}

// Processor interface for source-specific webhook processing
type Processor interface {
	ProcessWebhook(payload []byte, headers http.Header) (*types.LiberationGuardianEvent, error)
	ValidateSignature(payload []byte, signature string, secret string) bool
	GetEventSource() types.EventSource
}

// NewReceiver creates a new webhook receiver
func NewReceiver(cfg *config.Config, logger *logrus.Logger, eventChan chan *types.LiberationGuardianEvent) *Receiver {
	r := &Receiver{
		config:     cfg,
		logger:     logger,
		eventChan:  eventChan,
		processors: make(map[types.EventSource]Processor),
	}

	// Register processors for different sources
	r.registerProcessors()

	return r
}

// registerProcessors registers webhook processors for different sources
func (r *Receiver) registerProcessors() {
	if r.config.Integrations.Observability.Sentry.Enabled {
		r.processors[types.SourceSentry] = NewSentryProcessor(r.logger)
	}
	if r.config.Integrations.Observability.Prometheus.Enabled {
		r.processors[types.SourcePrometheus] = NewPrometheusProcessor(r.logger)
	}
	if r.config.Integrations.Observability.Grafana.Enabled {
		r.processors[types.SourceGrafana] = NewGrafanaProcessor(r.logger)
	}
	if r.config.Integrations.SourceControl.GitHub.Enabled {
		r.processors[types.SourceGitHub] = NewGitHubProcessor(r.logger)
	}
}

// SetupRoutes configures webhook routes
func (r *Receiver) SetupRoutes(router *gin.Engine) {
	webhooks := router.Group("/webhook")

	// Universal webhook endpoint - auto-detects source
	webhooks.POST("/", r.handleUniversalWebhook)

	// Source-specific endpoints
	webhooks.POST("/sentry", r.handleSourceWebhook(types.SourceSentry))
	webhooks.POST("/prometheus", r.handleSourceWebhook(types.SourcePrometheus))
	webhooks.POST("/grafana", r.handleSourceWebhook(types.SourceGrafana))
	webhooks.POST("/github", r.handleSourceWebhook(types.SourceGitHub))
	webhooks.POST("/gitlab", r.handleSourceWebhook(types.SourceGitLab))

	// Custom webhook endpoint
	webhooks.POST("/custom/:source", r.handleCustomWebhook)
}

// handleUniversalWebhook attempts to auto-detect the source and process accordingly
func (r *Receiver) handleUniversalWebhook(c *gin.Context) {
	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		r.logger.Errorf("Failed to read webhook payload: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read payload"})
		return
	}

	// Auto-detect source based on headers and payload structure
	source := r.detectSource(c.Request.Header, payload)
	if source == "" {
		r.logger.Warn("Could not auto-detect webhook source")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Could not detect webhook source"})
		return
	}

	r.processWebhook(c, source, payload)
}

// handleSourceWebhook handles webhooks for a specific source
func (r *Receiver) handleSourceWebhook(source types.EventSource) gin.HandlerFunc {
	return func(c *gin.Context) {
		payload, err := io.ReadAll(c.Request.Body)
		if err != nil {
			r.logger.Errorf("Failed to read webhook payload: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read payload"})
			return
		}

		r.processWebhook(c, source, payload)
	}
}

// handleCustomWebhook handles custom webhook sources
func (r *Receiver) handleCustomWebhook(c *gin.Context) {
	source := types.EventSource(c.Param("source"))

	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		r.logger.Errorf("Failed to read webhook payload: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read payload"})
		return
	}

	// For custom sources, create a generic event
	event := r.createGenericEvent(source, payload, c.Request.Header)

	// Send to processing pipeline
	select {
	case r.eventChan <- event:
		r.logger.Infof("Custom webhook event queued: %s from %s", event.ID, source)
	default:
		r.logger.Error("Event channel full, dropping event")
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "System overloaded"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "received", "event_id": event.ID})
}

// processWebhook processes a webhook for a specific source
func (r *Receiver) processWebhook(c *gin.Context, source types.EventSource, payload []byte) {
	processor, exists := r.processors[source]
	if !exists {
		r.logger.Errorf("No processor registered for source: %s", source)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported webhook source"})
		return
	}

	// Validate webhook signature if configured
	if !r.validateWebhookSignature(c.Request.Header, payload, source) {
		r.logger.Warnf("Invalid webhook signature for source: %s", source)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid signature"})
		return
	}

	// Process the webhook
	event, err := processor.ProcessWebhook(payload, c.Request.Header)
	if err != nil {
		r.logger.Errorf("Failed to process webhook from %s: %v", source, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to process webhook"})
		return
	}

	// Send to processing pipeline
	select {
	case r.eventChan <- event:
		r.logger.Infof("Webhook event queued: %s from %s", event.ID, source)
	default:
		r.logger.Error("Event channel full, dropping event")
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "System overloaded"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "received", "event_id": event.ID})
}

// detectSource attempts to auto-detect the webhook source
func (r *Receiver) detectSource(headers http.Header, payload []byte) types.EventSource {
	// Check User-Agent header
	userAgent := headers.Get("User-Agent")
	switch {
	case strings.Contains(userAgent, "Sentry"):
		return types.SourceSentry
	case strings.Contains(userAgent, "Prometheus"):
		return types.SourcePrometheus
	case strings.Contains(userAgent, "GitHub"):
		return types.SourceGitHub
	case strings.Contains(userAgent, "GitLab"):
		return types.SourceGitLab
	}

	// Check for source-specific headers
	if headers.Get("X-Sentry-Hook-Resource") != "" {
		return types.SourceSentry
	}
	if headers.Get("X-GitHub-Event") != "" {
		return types.SourceGitHub
	}
	if headers.Get("X-Gitlab-Event") != "" {
		return types.SourceGitLab
	}

	// Try to detect from payload structure
	var jsonPayload map[string]interface{}
	if err := json.Unmarshal(payload, &jsonPayload); err == nil {
		if _, exists := jsonPayload["sentry"]; exists {
			return types.SourceSentry
		}
		if _, exists := jsonPayload["alerts"]; exists {
			return types.SourcePrometheus
		}
		if _, exists := jsonPayload["repository"]; exists {
			return types.SourceGitHub
		}
	}

	return ""
}

// validateWebhookSignature validates the webhook signature
func (r *Receiver) validateWebhookSignature(headers http.Header, payload []byte, source types.EventSource) bool {
	secret := r.config.GetWebhookSecret(string(source))
	if secret == "" {
		// No secret configured, skip validation
		return true
	}

	processor, exists := r.processors[source]
	if !exists {
		return false
	}

	signature := r.extractSignature(headers, source)
	if signature == "" {
		return false
	}

	return processor.ValidateSignature(payload, signature, secret)
}

// extractSignature extracts the signature from headers based on source
func (r *Receiver) extractSignature(headers http.Header, source types.EventSource) string {
	switch source {
	case types.SourceSentry:
		return headers.Get("Sentry-Hook-Signature")
	case types.SourceGitHub:
		return headers.Get("X-Hub-Signature-256")
	case types.SourceGitLab:
		return headers.Get("X-Gitlab-Token")
	case types.SourceGrafana:
		return headers.Get("Authorization")
	default:
		return ""
	}
}

// createGenericEvent creates a generic event for unknown sources
func (r *Receiver) createGenericEvent(source types.EventSource, payload []byte, headers http.Header) *types.LiberationGuardianEvent {
	event := &types.LiberationGuardianEvent{
		ID:          uuid.New().String(),
		Source:      string(source),
		Type:        "webhook",
		Severity:    types.SeverityMedium,
		Timestamp:   time.Now(),
		Title:       fmt.Sprintf("Webhook from %s", source),
		Description: "Generic webhook event",
		RawPayload:  json.RawMessage(payload),
		Metadata:    make(map[string]interface{}),
		Environment: r.config.Core.Environment,
		Tags:        []string{"webhook", "custom"},
	}

	// Add relevant headers to metadata
	for key, values := range headers {
		if len(values) > 0 {
			event.Metadata[fmt.Sprintf("header_%s", strings.ToLower(key))] = values[0]
		}
	}

	// Generate fingerprint for deduplication
	event.Fingerprint = r.generateFingerprint(event)

	return event
}

// generateFingerprint generates a fingerprint for event deduplication
func (r *Receiver) generateFingerprint(event *types.LiberationGuardianEvent) string {
	data := fmt.Sprintf("%s:%s:%s:%s", event.Source, event.Type, event.Title, event.Service)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])[:16] // Use first 16 chars
}

// ValidateHMAC validates HMAC signature
func ValidateHMAC(payload []byte, signature, secret string) bool {
	// Remove prefix if present (e.g., "sha256=")
	signature = strings.TrimPrefix(signature, "sha256=")

	expectedSig, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}

	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(payload) // hash.Hash.Write never returns an error
	actualSig := mac.Sum(nil)

	return hmac.Equal(expectedSig, actualSig)
}
