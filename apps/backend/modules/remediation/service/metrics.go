package service

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	remediationActionsFailedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "remediation_actions_failed_total",
		Help: "Total number of failed remediation actions",
	}, []string{"action_type", "failure_reason"})

	remediationLineageDriftTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "remediation_lineage_drift_total",
		Help: "Total remediation actions completed but Neo4j lineage sync failed",
	})

	remediationBatchPartialTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "remediation_batch_partial_total",
		Help: "Total batch remediation requests that completed with partial failures",
	})
)
