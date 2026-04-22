package orchestrator

import (
	"context"
	"fmt"
	"time"

	"github.com/arc-platform/go-scanner/internal/presidio"
	"github.com/sony/gobreaker"
)

// presidioBreaker trips after 3 consecutive Presidio failures and stays open
// for 30 seconds before allowing a single probe request.
var presidioBreaker = gobreaker.NewCircuitBreaker(gobreaker.Settings{
	Name:        "presidio",
	MaxRequests: 1,
	Interval:    60 * time.Second,
	Timeout:     30 * time.Second,
	ReadyToTrip: func(counts gobreaker.Counts) bool {
		return counts.ConsecutiveFailures >= 3
	},
	OnStateChange: func(name string, from, to gobreaker.State) {
		fmt.Printf("circuit breaker %q: %s → %s\n", name, from, to)
	},
})

// analyzeWithBreaker runs Presidio analysis through the circuit breaker.
// When the breaker is open (Presidio consistently unavailable), returns nil
// immediately so the scanner falls back to regex-only detection.
//
// Error detection: presidio.Client.Analyze swallows HTTP errors internally.
// We use client.Health as the failure signal when Analyze returns nil, which
// distinguishes "no PII found" from "Presidio is down". The health call is
// only made on a nil result so the happy-path (entities found) has no overhead.
func analyzeWithBreaker(ctx context.Context, client *presidio.Client, value string, opts presidio.AnalyzeOptions) []presidio.Entity {
	result, err := presidioBreaker.Execute(func() (interface{}, error) {
		entities := client.Analyze(ctx, value, opts)
		if entities == nil {
			// nil could mean "no PII" or "request failed" — use Health to distinguish.
			if hErr := client.Health(ctx); hErr != nil {
				return nil, hErr // counts as a failure toward the breaker threshold
			}
		}
		return entities, nil
	})
	if err != nil {
		// Breaker open or Presidio unreachable — degrade gracefully.
		return nil
	}
	if result == nil {
		return nil
	}
	return result.([]presidio.Entity)
}
