package orchestrator

import (
	"context"
	"log"

	"github.com/arc-platform/go-scanner/internal/classifier"
	"github.com/arc-platform/go-scanner/internal/connectors"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

// ScanConfig is the input to RunScan.
type ScanConfig struct {
	ScanID         string
	Sources        []SourceSpec
	CustomPatterns []classifier.CustomPattern
	MaxConcurrency int
	BackendURL     string
}

// SourceSpec describes one data source to scan.
type SourceSpec struct {
	SourceType string
	Config     map[string]any
}

// Orchestrator coordinates parallel scanning across multiple sources.
type Orchestrator struct {
	engine *classifier.Engine
}

// NewOrchestrator creates a new Orchestrator.
func NewOrchestrator() *Orchestrator {
	return &Orchestrator{engine: classifier.NewEngine()}
}

// RunScan scans all sources in parallel (up to MaxConcurrency) and returns all findings.
func (o *Orchestrator) RunScan(ctx context.Context, cfg ScanConfig) ([]classifier.ClassifiedFinding, error) {
	if cfg.MaxConcurrency <= 0 {
		cfg.MaxConcurrency = 8
	}

	g, gctx := errgroup.WithContext(ctx)
	sem := semaphore.NewWeighted(int64(cfg.MaxConcurrency))

	resultsCh := make(chan []classifier.ClassifiedFinding, len(cfg.Sources)+1)

	for _, src := range cfg.Sources {
		src := src
		g.Go(func() error {
			if err := sem.Acquire(gctx, 1); err != nil {
				return err
			}
			defer sem.Release(1)

			findings, err := o.scanSource(gctx, src, cfg.CustomPatterns)
			if err != nil {
				log.Printf("WARN: source %s scan failed: %v", src.SourceType, err)
				return nil // non-fatal: continue other sources
			}
			if len(findings) > 0 {
				resultsCh <- findings
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}
	close(resultsCh)

	var all []classifier.ClassifiedFinding
	for batch := range resultsCh {
		all = append(all, batch...)
	}
	return all, nil
}

func (o *Orchestrator) scanSource(ctx context.Context, src SourceSpec, custom []classifier.CustomPattern) ([]classifier.ClassifiedFinding, error) {
	conn, err := connectors.Dispatch(src.SourceType)
	if err != nil {
		return nil, err
	}
	if err := conn.Connect(ctx, src.Config); err != nil {
		return nil, err
	}
	defer conn.Close()

	fieldsCh, errCh := conn.StreamFields(ctx)

	var findings []classifier.ClassifiedFinding
	for {
		select {
		case rec, ok := <-fieldsCh:
			if !ok {
				return findings, nil
			}
			batch := o.engine.Classify(rec, custom)
			findings = append(findings, batch...)
		case err, ok := <-errCh:
			if ok && err != nil {
				return findings, err
			}
		case <-ctx.Done():
			return findings, ctx.Err()
		}
	}
}
