package classifier

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"github.com/arc-platform/go-scanner/internal/connectors"
)

// sampleText is a ~1 KB corpus containing representative PII patterns used to
// exercise the full classify pipeline realistically.
var sampleText = strings.Repeat(
	"User ABCPE1234F submitted KYC. Bank account 123456789012 linked. "+
		"Email: alice.smith@example.com. IFSC: SBIN0001234. "+
		"Aadhaar: 234123412346. Passport A1234567. Phone +919876543210. ",
	8, // ~800 bytes
)

// BenchmarkClassifyText measures classification throughput on a ~1 KB field value.
func BenchmarkClassifyText(b *testing.B) {
	eng := NewEngine()
	rec := connectors.FieldRecord{
		Value:      sampleText,
		FieldName:  "kyc_data",
		SourcePath: "customers.kyc",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = eng.Classify(rec, nil, nil)
	}
}

// BenchmarkClassifyText_AllowedSubset benchmarks with an explicit allow-list
// (only EMAIL_ADDRESS + IN_PAN) to measure early-exit path.
func BenchmarkClassifyText_AllowedSubset(b *testing.B) {
	eng := NewEngine()
	rec := connectors.FieldRecord{
		Value:     sampleText,
		FieldName: "kyc_data",
	}
	allowed := map[string]struct{}{
		"Email Address": {},
		"PAN":           {},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = eng.Classify(rec, nil, allowed)
	}
}

// ─── Reservoir sampling ───────────────────────────────────────────────────────
//
// Two common strategies for sampling k rows out of n:
//   A. ORDER BY random() LIMIT k  — full sort, O(n log n)
//   B. Reservoir sampling (Vitter R) — single pass, O(n)
//
// The benchmarks below simulate these in memory using []string rows so the
// comparison is pure algorithmic cost, independent of DB I/O.

func makeRows(n int) []string {
	rows := make([]string, n)
	for i := range rows {
		rows[i] = fmt.Sprintf("row-%07d", i)
	}
	return rows
}

// sortSample simulates ORDER BY random() LIMIT k: shuffle a copy, take first k.
func sortSample(rows []string, k int) []string {
	cp := make([]string, len(rows))
	copy(cp, rows)
	rand.Shuffle(len(cp), func(i, j int) { cp[i], cp[j] = cp[j], cp[i] })
	if k > len(cp) {
		k = len(cp)
	}
	return cp[:k]
}

// reservoirSample implements Vitter Algorithm R: single pass, O(n) time, O(k) space.
func reservoirSample(rows []string, k int) []string {
	if k >= len(rows) {
		return rows
	}
	reservoir := make([]string, k)
	copy(reservoir, rows[:k])
	for i := k; i < len(rows); i++ {
		j := rand.Intn(i + 1)
		if j < k {
			reservoir[j] = rows[i]
		}
	}
	return reservoir
}

const (
	benchN = 10_000
	benchK = 100
)

var benchRows = makeRows(benchN)

func BenchmarkReservoirSampling_SortApproach(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = sortSample(benchRows, benchK)
	}
}

func BenchmarkReservoirSampling_VitterR(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = reservoirSample(benchRows, benchK)
	}
}
