package events

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"

	"liberation-guardian/pkg/types"
)

// RedisKnowledgeBase implements KnowledgeBase using Redis
type RedisKnowledgeBase struct {
	client *redis.Client
	logger *logrus.Logger
}

// NewRedisKnowledgeBase creates a new Redis-based knowledge base
func NewRedisKnowledgeBase(client *redis.Client, logger *logrus.Logger) *RedisKnowledgeBase {
	return &RedisKnowledgeBase{
		client: client,
		logger: logger,
	}
}

// FindSimilarPatterns finds patterns similar to the given event
func (kb *RedisKnowledgeBase) FindSimilarPatterns(ctx context.Context, event *types.LiberationGuardianEvent) ([]*types.KnowledgePattern, error) {
	// For now, implement a simple pattern matching based on event fingerprint and source
	// In a full implementation, this would use vector similarity search

	patterns := []*types.KnowledgePattern{}

	// Search for patterns by source and type
	searchKey := fmt.Sprintf("patterns:%s:%s", event.Source, event.Type)

	patternIDs, err := kb.client.SMembers(ctx, searchKey).Result()
	if err != nil {
		kb.logger.Debugf("No patterns found for key %s: %v", searchKey, err)
		return patterns, nil // Return empty on error
	}

	for _, patternID := range patternIDs {
		pattern, err := kb.getPattern(ctx, patternID)
		if err != nil {
			continue
		}
		patterns = append(patterns, pattern)
	}

	return patterns, nil
}

// RecordResolution records the outcome of a resolution attempt
func (kb *RedisKnowledgeBase) RecordResolution(ctx context.Context, eventID string, resolution *types.AutoFixPlan, success bool) error {
	resolutionKey := fmt.Sprintf("resolutions:%s", eventID)

	resolutionData := map[string]interface{}{
		"event_id":   eventID,
		"resolution": resolution,
		"success":    success,
		"timestamp":  time.Now(),
	}

	jsonData, err := json.Marshal(resolutionData)
	if err != nil {
		return err
	}

	return kb.client.Set(ctx, resolutionKey, jsonData, 30*24*time.Hour).Err() // Keep for 30 days
}

// UpdatePatternConfidence updates the confidence score of a pattern
func (kb *RedisKnowledgeBase) UpdatePatternConfidence(ctx context.Context, patternID string, feedback float64) error {
	// Get current pattern
	pattern, err := kb.getPattern(ctx, patternID)
	if err != nil {
		return err
	}

	// Update confidence with exponential moving average
	alpha := 0.1 // Learning rate
	pattern.Confidence = pattern.Confidence*(1-alpha) + feedback*alpha
	pattern.LastSeen = time.Now()

	// Save updated pattern
	return kb.savePattern(ctx, pattern)
}

// getPattern retrieves a pattern by ID
func (kb *RedisKnowledgeBase) getPattern(ctx context.Context, patternID string) (*types.KnowledgePattern, error) {
	patternKey := fmt.Sprintf("pattern:%s", patternID)

	data, err := kb.client.Get(ctx, patternKey).Result()
	if err != nil {
		return nil, err
	}

	var pattern types.KnowledgePattern
	if err := json.Unmarshal([]byte(data), &pattern); err != nil {
		return nil, err
	}

	return &pattern, nil
}

// savePattern saves a pattern to Redis
func (kb *RedisKnowledgeBase) savePattern(ctx context.Context, pattern *types.KnowledgePattern) error {
	patternKey := fmt.Sprintf("pattern:%s", pattern.ID)

	jsonData, err := json.Marshal(pattern)
	if err != nil {
		return err
	}

	return kb.client.Set(ctx, patternKey, jsonData, 0).Err() // No expiration
}
