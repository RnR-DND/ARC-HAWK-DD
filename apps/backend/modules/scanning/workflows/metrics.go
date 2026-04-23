package workflows

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	temporalWorkflowFailuresTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "temporal_workflow_failures_total",
		Help: "Total Temporal workflow failures by workflow type",
	}, []string{"workflow_type"})

	temporalActivityFailuresTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "temporal_activity_failures_total",
		Help: "Total Temporal activity failures by activity name",
	}, []string{"activity_name"})
)

// RecordWorkflowFailure increments the workflow failure counter.
func RecordWorkflowFailure(workflowType string) {
	temporalWorkflowFailuresTotal.WithLabelValues(workflowType).Inc()
}

// RecordActivityFailure increments the activity failure counter.
func RecordActivityFailure(activityName string) {
	temporalActivityFailuresTotal.WithLabelValues(activityName).Inc()
}
