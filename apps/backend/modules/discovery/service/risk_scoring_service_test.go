package service

import (
	"testing"

	"github.com/arc-platform/backend/modules/discovery/domain"
)

// TestComputeScore is a table-driven test for the pure scoring function.
// These tests run without a database — fast and deterministic.
func TestComputeScore(t *testing.T) {
	defaults := domain.DefaultRiskWeights()

	tests := []struct {
		name    string
		rows    []domain.InventoryRow
		weights domain.RiskWeights
		want    float64
	}{
		{
			name:    "empty slice returns zero",
			rows:    nil,
			weights: defaults,
			want:    0,
		},
		{
			name: "zero findings on a single row returns zero contribution from volume",
			rows: []domain.InventoryRow{
				{FindingCount: 0, Sensitivity: 50},
			},
			weights: defaults,
			// volume contrib = 0 * 1 = 0
			// sensitivity contrib = 50 * 2 = 100
			// cross-product = 0 * 50 * 1.5 = 0
			want: 100,
		},
		{
			name: "single high-sensitivity row",
			rows: []domain.InventoryRow{
				{FindingCount: 10, Sensitivity: 100},
			},
			weights: defaults,
			// 10*1 + 100*2 + 10*100*1*1.5 = 10 + 200 + 1500 = 1710
			want: 1710,
		},
		{
			name: "negative inputs are clamped to zero",
			rows: []domain.InventoryRow{
				{FindingCount: -5, Sensitivity: -10},
			},
			weights: defaults,
			want:    0,
		},
		{
			name: "multiple rows additive",
			rows: []domain.InventoryRow{
				{FindingCount: 1, Sensitivity: 10},
				{FindingCount: 2, Sensitivity: 20},
			},
			weights: defaults,
			// row 1: 1 + 20 + 1*10*1.5 = 1 + 20 + 15 = 36
			// row 2: 2 + 40 + 2*20*1.5 = 2 + 40 + 60 = 102
			// total = 138
			want: 138,
		},
		{
			name: "custom weights override",
			rows: []domain.InventoryRow{
				{FindingCount: 10, Sensitivity: 50},
			},
			weights: domain.RiskWeights{Volume: 0.5, Sensitivity: 1.0, Exposure: 0.5},
			// 10*0.5 + 50*1.0 + 10*50*1*0.5 = 5 + 50 + 250 = 305
			want: 305,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeScore(tt.rows, tt.weights)
			if got != tt.want {
				t.Errorf("ComputeScore() = %v, want %v", got, tt.want)
			}
		})
	}
}
