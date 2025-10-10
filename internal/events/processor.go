package events

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"

	"liberation-guardian/internal/ai"
	"liberation-guardian/internal/codebase"
	"liberation-guardian/internal/config"
	"liberation-guardian/pkg/types"
)

// Processor handles Liberation Guardian events and integrates with The Collective Strategist event system
type Processor struct {
	config       *config.Config
	logger       *logrus.Logger
	aiClient     ai.AIClient
	redisClient  *redis.Client
	triageEngine *ai.TriageEngine
}

// NewProcessor creates a new event processor
func NewProcessor(cfg *config.Config, logger *logrus.Logger, aiClient ai.AIClient) (*Processor, error) {
	// Connect to Redis (same instance as The Collective Strategist)
	redisClient := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	// Create knowledge base (simple Redis-based implementation for now)
	knowledgeBase := NewRedisKnowledgeBase(redisClient, logger)

	// Create triage engine
	// Initialize codebase analyzer
	codeAnalyzerConfig := &codebase.AnalyzerConfig{
		AllowedPaths:      []string{"src/", "internal/", "pkg/", "lib/", "app/", "services/"},
		BlockedPaths:      []string{".env", ".secret", ".git/", "node_modules/", "vendor/"},
		MaxFileSize:       100 * 1024, // 100KB
		MaxFiles:          20,
		IncludeGitHistory: true,
		MaxCommitHistory:  10,
		TrustLevel:        "cautious",
	}

	codebaseAnalyzer, err := codebase.NewCodebaseAnalyzer(logger, ".", codeAnalyzerConfig)
	if err != nil {
		logger.Warnf("Failed to initialize codebase analyzer: %v", err)
		codebaseAnalyzer = nil // Continue without codebase analysis
	}

	triageEngine := ai.NewTriageEngine(cfg, logger, aiClient, knowledgeBase, codebaseAnalyzer)

	return &Processor{
		config:       cfg,
		logger:       logger,
		aiClient:     aiClient,
		redisClient:  redisClient,
		triageEngine: triageEngine,
	}, nil
}

// ProcessEvent processes a Liberation Guardian event
func (p *Processor) ProcessEvent(ctx context.Context, event *types.LiberationGuardianEvent) error {
	p.logger.Infof("Processing event %s from %s", event.ID, event.Source)

	// Step 1: Perform AI triage
	triageResult, err := p.triageEngine.TriageEvent(ctx, event)
	if err != nil {
		p.logger.Errorf("Triage failed for event %s: %v", event.ID, err)
		// Fallback: escalate to human
		return p.escalateToHuman(ctx, event, fmt.Sprintf("Triage failed: %v", err))
	}

	// Step 2: Execute the triage decision
	switch triageResult.Decision {
	case types.DecisionAutoAcknowledge:
		return p.autoAcknowledge(ctx, event, triageResult)
	case types.DecisionAutoFix:
		return p.attemptAutoFix(ctx, event, triageResult)
	case types.DecisionEscalateHuman:
		return p.escalateToHuman(ctx, event, triageResult.Reasoning)
	case types.DecisionAnalyzeDeeper:
		return p.analyzeDeeper(ctx, event, triageResult)
	case types.DecisionIgnore:
		return p.ignoreEvent(ctx, event, triageResult)
	default:
		return p.escalateToHuman(ctx, event, "Unknown triage decision")
	}
}

// autoAcknowledge handles auto-acknowledged events
func (p *Processor) autoAcknowledge(ctx context.Context, event *types.LiberationGuardianEvent, result *types.TriageResult) error {
	p.logger.Infof("Auto-acknowledging event %s: %s", event.ID, result.Reasoning)

	// Publish to The Collective Strategist event system
	return p.publishCollectiveStrategistEvent(ctx, map[string]interface{}{
		"stream":         "system.events",
		"type":           "liberation_guardian.event.auto_acknowledged",
		"version":        1,
		"user_id":        nil,
		"correlation_id": event.CorrelationID,
		"data": map[string]interface{}{
			"liberation_event_id":  event.ID,
			"source":               event.Source,
			"original_type":        event.Type,
			"triage_decision":      result.Decision,
			"triage_confidence":    result.Confidence,
			"triage_reasoning":     result.Reasoning,
			"auto_acknowledged_at": time.Now(),
		},
	})
}

// attemptAutoFix handles auto-fix attempts
func (p *Processor) attemptAutoFix(ctx context.Context, event *types.LiberationGuardianEvent, result *types.TriageResult) error {
	p.logger.Infof("Attempting auto-fix for event %s: %s", event.ID, result.Reasoning)

	if result.AutoFixAttempt == nil {
		return p.escalateToHuman(ctx, event, "No auto-fix plan provided")
	}

	// For now, just publish the auto-fix attempt
	// In a full implementation, this would execute the fix steps
	return p.publishCollectiveStrategistEvent(ctx, map[string]interface{}{
		"stream":         "system.events",
		"type":           "liberation_guardian.autofix.attempted",
		"version":        1,
		"user_id":        nil,
		"correlation_id": event.CorrelationID,
		"data": map[string]interface{}{
			"liberation_event_id": event.ID,
			"source":              event.Source,
			"original_type":       event.Type,
			"fix_plan":            result.AutoFixAttempt,
			"triage_confidence":   result.Confidence,
			"attempted_at":        time.Now(),
			"status":              "pending",
		},
	})
}

// escalateToHuman handles human escalation
func (p *Processor) escalateToHuman(ctx context.Context, event *types.LiberationGuardianEvent, reason string) error {
	p.logger.Warnf("Escalating event %s to human: %s", event.ID, reason)

	// Publish notification request to The Collective Strategist
	return p.publishCollectiveStrategistEvent(ctx, map[string]interface{}{
		"stream":         "notification.events",
		"type":           "notification.send.requested",
		"version":        1,
		"user_id":        nil, // Could be mapped to admin users
		"correlation_id": event.CorrelationID,
		"data": map[string]interface{}{
			"user_id":           nil, // Admin notification
			"notification_type": "system_alert",
			"channels":          []string{"email", "slack"},
			"message": map[string]interface{}{
				"title":      fmt.Sprintf("Liberation Guardian Alert: %s", event.Title),
				"body":       fmt.Sprintf("Event from %s requires human attention.\n\nReason: %s\n\nDescription: %s", event.Source, reason, event.Description),
				"action_url": fmt.Sprintf("/admin/events/%s", event.ID),
			},
			"priority":                "high",
			"liberation_event_id":     event.ID,
			"liberation_event_source": event.Source,
			"escalation_reason":       reason,
			"escalated_at":            time.Now(),
		},
	})
}

// analyzeDeeper handles deeper analysis requests
func (p *Processor) analyzeDeeper(ctx context.Context, event *types.LiberationGuardianEvent, result *types.TriageResult) error {
	p.logger.Infof("Requesting deeper analysis for event %s", event.ID)

	// This would typically invoke the Analysis Agent
	// For now, just log and escalate
	return p.escalateToHuman(ctx, event, "Deeper analysis requested but not yet implemented")
}

// ignoreEvent handles ignored events
func (p *Processor) ignoreEvent(ctx context.Context, event *types.LiberationGuardianEvent, result *types.TriageResult) error {
	p.logger.Debugf("Ignoring event %s: %s", event.ID, result.Reasoning)

	// Still log the decision for audit purposes
	return p.publishCollectiveStrategistEvent(ctx, map[string]interface{}{
		"stream":         "system.events",
		"type":           "liberation_guardian.event.ignored",
		"version":        1,
		"user_id":        nil,
		"correlation_id": event.CorrelationID,
		"data": map[string]interface{}{
			"liberation_event_id": event.ID,
			"source":              event.Source,
			"original_type":       event.Type,
			"triage_decision":     result.Decision,
			"triage_confidence":   result.Confidence,
			"triage_reasoning":    result.Reasoning,
			"ignored_at":          time.Now(),
		},
	})
}

// publishCollectiveStrategistEvent publishes an event to The Collective Strategist event system
func (p *Processor) publishCollectiveStrategistEvent(ctx context.Context, eventData map[string]interface{}) error {
	// Add standard fields
	eventData["id"] = p.generateEventID()
	eventData["timestamp"] = time.Now()

	// Convert to Redis stream format
	fields := make(map[string]interface{})
	for key, value := range eventData {
		if key == "data" {
			// Serialize complex data as JSON
			if jsonData, err := json.Marshal(value); err == nil {
				fields[key] = string(jsonData)
			}
		} else if value != nil {
			fields[key] = fmt.Sprintf("%v", value)
		}
	}

	// Publish to Redis stream
	streamName := "system.events" // Default stream
	if stream, ok := eventData["stream"].(string); ok {
		streamName = stream
	}

	_, err := p.redisClient.XAdd(ctx, &redis.XAddArgs{
		Stream: streamName,
		ID:     "*",
		Values: fields,
	}).Result()

	if err != nil {
		return fmt.Errorf("failed to publish event to stream %s: %w", streamName, err)
	}

	p.logger.Debugf("Published event to stream %s", streamName)
	return nil
}

// generateEventID generates a unique event ID
func (p *Processor) generateEventID() string {
	// Simple timestamp-based ID for now
	// In production, would use ULID or UUID
	return fmt.Sprintf("lg_%d", time.Now().UnixNano())
}
