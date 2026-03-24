package service

import (
	"context"
	"fmt"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// PIIExposure represents a PII exposure at a specific point in time
type PIIExposure struct {
	PIIType      string
	ExposedSince time.Time
	ExposedUntil *time.Time // nil if still exposed
	FirstScanID  string
	LastScanID   string
}

// ExposureWindow represents a complete exposure history for a PII type
type ExposureWindow struct {
	PIIType     string
	Since       time.Time
	Until       *time.Time
	FirstScanID string
	LastScanID  string
	IsActive    bool
}

// TemporalLineageService provides temporal-aware lineage queries
type TemporalLineageService struct {
	neo4j neo4j.Driver
}

// NewTemporalLineageService creates a new temporal lineage service
func NewTemporalLineageService(driver neo4j.Driver) *TemporalLineageService {
	return &TemporalLineageService{
		neo4j: driver,
	}
}

// GetExposureAtTime returns all PII exposed at a specific point in time
func (s *TemporalLineageService) GetExposureAtTime(ctx context.Context, assetID string, atTime time.Time) ([]PIIExposure, error) {
	session := s.neo4j.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	result, err := session.Run(`
		MATCH (a:Asset {stable_id: $assetID})-[e:EXPOSES]->(p:PII_Category)
		WHERE e.since <= $atTime 
		  AND (e.until IS NULL OR e.until > $atTime)
		RETURN p.pii_type as pii_type, 
		       e.since as exposed_since,
		       e.until as exposed_until,
		       e.first_scan_id as first_scan_id,
		       e.last_scan_id as last_scan_id
	`, map[string]interface{}{
		"assetID": assetID,
		"atTime":  atTime,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query exposure at time: %w", err)
	}

	var exposures []PIIExposure
	for result.Next() {
		record := result.Record()

		var exposedUntil *time.Time
		if untilVal, ok := record.Get("exposed_until"); ok && untilVal != nil {
			if t, ok := untilVal.(time.Time); ok {
				exposedUntil = &t
			}
		}

		exposures = append(exposures, PIIExposure{
			PIIType:      record.Values[0].(string),
			ExposedSince: record.Values[1].(time.Time),
			ExposedUntil: exposedUntil,
			FirstScanID:  record.Values[3].(string),
			LastScanID:   record.Values[4].(string),
		})
	}

	return exposures, nil
}

// GetExposureHistory returns complete exposure history for an asset
func (s *TemporalLineageService) GetExposureHistory(ctx context.Context, assetID string) ([]ExposureWindow, error) {
	session := s.neo4j.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	result, err := session.Run(`
		MATCH (a:Asset {stable_id: $assetID})-[e:EXPOSES]->(p:PII_Category)
		RETURN p.pii_type as pii_type,
		       e.since as since,
		       e.until as until,
		       e.first_scan_id as first_scan,
		       e.last_scan_id as last_scan
		ORDER BY e.since DESC
	`, map[string]interface{}{
		"assetID": assetID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query exposure history: %w", err)
	}

	var windows []ExposureWindow
	for result.Next() {
		record := result.Record()

		var until *time.Time
		if untilVal, ok := record.Get("until"); ok && untilVal != nil {
			if t, ok := untilVal.(time.Time); ok {
				until = &t
			}
		}

		windows = append(windows, ExposureWindow{
			PIIType:     record.Values[0].(string),
			Since:       record.Values[1].(time.Time),
			Until:       until,
			FirstScanID: record.Values[3].(string),
			LastScanID:  record.Values[4].(string),
			IsActive:    until == nil,
		})
	}

	return windows, nil
}

// GetActiveExposures returns all currently active PII exposures for an asset
func (s *TemporalLineageService) GetActiveExposures(ctx context.Context, assetID string) ([]PIIExposure, error) {
	session := s.neo4j.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	result, err := session.Run(`
		MATCH (a:Asset {stable_id: $assetID})-[e:EXPOSES]->(p:PII_Category)
		WHERE e.until IS NULL
		RETURN p.pii_type as pii_type,
		       e.since as exposed_since,
		       e.until as exposed_until,
		       e.first_scan_id as first_scan_id,
		       e.last_scan_id as last_scan_id
	`, map[string]interface{}{
		"assetID": assetID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query active exposures: %w", err)
	}

	var exposures []PIIExposure
	for result.Next() {
		record := result.Record()
		exposures = append(exposures, PIIExposure{
			PIIType:      record.Values[0].(string),
			ExposedSince: record.Values[1].(time.Time),
			ExposedUntil: nil, // Active exposures have no end time
			FirstScanID:  record.Values[3].(string),
			LastScanID:   record.Values[4].(string),
		})
	}

	return exposures, nil
}

// GetExposureDuration calculates how long a PII type was exposed
func (s *TemporalLineageService) GetExposureDuration(ctx context.Context, assetID string, piiType string) (time.Duration, error) {
	session := s.neo4j.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	result, err := session.Run(`
		MATCH (a:Asset {stable_id: $assetID})-[e:EXPOSES]->(p:PII_Category {pii_type: $piiType})
		RETURN e.since as since, e.until as until
	`, map[string]interface{}{
		"assetID": assetID,
		"piiType": piiType,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to query exposure duration: %w", err)
	}

	if !result.Next() {
		return 0, fmt.Errorf("no exposure found for PII type %s", piiType)
	}

	record := result.Record()
	since := record.Values[0].(time.Time)

	var until time.Time
	if untilVal := record.Values[1]; untilVal != nil {
		until = untilVal.(time.Time)
	} else {
		until = time.Now() // Still exposed, use current time
	}

	return until.Sub(since), nil
}

// WasCompliantAt checks if an asset was compliant at a specific time
// An asset is compliant if it had no active PII exposures at that time
func (s *TemporalLineageService) WasCompliantAt(ctx context.Context, assetID string, atTime time.Time) (bool, error) {
	exposures, err := s.GetExposureAtTime(ctx, assetID, atTime)
	if err != nil {
		return false, err
	}

	// Asset is compliant if it has no exposures
	return len(exposures) == 0, nil
}

// GetComplianceTimeline returns a timeline of compliance status changes
func (s *TemporalLineageService) GetComplianceTimeline(ctx context.Context, assetID string) ([]ComplianceEvent, error) {
	session := s.neo4j.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	result, err := session.Run(`
		MATCH (a:Asset {stable_id: $assetID})-[e:EXPOSES]->(p:PII_Category)
		RETURN e.since as event_time, 'EXPOSURE_STARTED' as event_type, p.pii_type as pii_type
		UNION
		MATCH (a:Asset {stable_id: $assetID})-[e:EXPOSES]->(p:PII_Category)
		WHERE e.until IS NOT NULL
		RETURN e.until as event_time, 'EXPOSURE_ENDED' as event_type, p.pii_type as pii_type
		ORDER BY event_time ASC
	`, map[string]interface{}{
		"assetID": assetID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query compliance timeline: %w", err)
	}

	var events []ComplianceEvent
	for result.Next() {
		record := result.Record()
		events = append(events, ComplianceEvent{
			EventTime: record.Values[0].(time.Time),
			EventType: record.Values[1].(string),
			PIIType:   record.Values[2].(string),
		})
	}

	return events, nil
}

// ComplianceEvent represents a change in compliance status
type ComplianceEvent struct {
	EventTime time.Time
	EventType string // 'EXPOSURE_STARTED' or 'EXPOSURE_ENDED'
	PIIType   string
}
