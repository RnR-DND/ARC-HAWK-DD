# ARC-HAWK-DD Benchmarks

**Platform:** darwin/arm64 · Apple M4  
**Go version:** see `go.mod`  
**Run:** `go test -bench=. -benchmem -benchtime=3s ./apps/goScanner/internal/classifier/...`

---

## PII Classification (`internal/classifier`)

| Benchmark | ns/op | B/op | allocs/op | Notes |
|-----------|------:|-----:|----------:|-------|
| `BenchmarkClassifyText` (full, ~1 KB) | 550,128 | 123,669 | 2,388 | All built-in patterns, ~800-byte corpus |
| `BenchmarkClassifyText_AllowedSubset` | 19,802 | 7,641 | 86 | Only EMAIL_ADDRESS + PAN — 28× faster |

**Takeaway:** filtering to an explicit `pii_types` allowlist cuts classification time by ~28× and allocations by ~28×. Always pass `allowedPatterns` when the scan job specifies `pii_types`.

---

## Reservoir Sampling (`internal/classifier`)

Benchmark compares two strategies for sampling *k=100* rows from *n=10,000*:

| Strategy | ns/op | B/op | allocs/op |
|----------|------:|-----:|----------:|
| `SortApproach` (shuffle full slice, take first k) | 72,555 | 163,840 | 1 |
| `VitterR` (single-pass reservoir, Algorithm R) | 65,941 | 1,792 | 1 |

**Takeaway:** Vitter R is ~9% faster and uses **91× less memory** (1.75 KB vs 160 KB). At scan scale (millions of rows) the memory advantage dominates — use reservoir sampling in connectors that implement row sub-sampling.

---

## How to reproduce

```bash
cd apps/goScanner
go test -bench=. -benchmem -benchtime=3s ./internal/classifier/...
```

To run a single benchmark:

```bash
go test -bench=BenchmarkClassifyText -benchmem ./internal/classifier/
```
