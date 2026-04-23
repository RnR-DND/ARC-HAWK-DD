package workflows

import (
	"fmt"
	"sort"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// StreamingSource identifies a Kafka topic or Kinesis stream being monitored.
type StreamingSource struct {
	SourceType    string `json:"source_type"`     // "kafka" | "kinesis"
	ConnectionKey string `json:"connection_key"`  // key in connection.yml
	TopicOrStream string `json:"topic_or_stream"` // topic name or stream name
	ProfileName   string `json:"profile_name"`
}

// StreamingCheckpointState carries the offset/sequence for a streaming source
// across Temporal workflow iterations.
type StreamingCheckpointState struct {
	Source        StreamingSource `json:"source"`
	KafkaOffset   int64           `json:"kafka_offset,omitempty"`    // last committed offset (per partition: simplified)
	KinesisSeqNum string          `json:"kinesis_seq_num,omitempty"` // last read sequence number
	WindowsRun    int             `json:"windows_run"`               // total micro-batch windows completed
	FindingsTotal int             `json:"findings_total"`            // cumulative findings emitted
}

// StreamingSupervisorInput is the workflow input.
type StreamingSupervisorInput struct {
	Sources          []StreamingSource `json:"sources"`
	WindowSeconds    int               `json:"window_seconds"`      // micro-batch window duration (default: 60)
	CheckpointEvery  int               `json:"checkpoint_every"`    // checkpoint every N windows (default: 5)
	MaxWindowsPerRun int               `json:"max_windows_per_run"` // continueAsNew after this many (default: 100)
	IngestEndpoint   string            `json:"ingest_endpoint"`     // backend ingest URL
	ScanProfileName  string            `json:"scan_profile_name"`
}

const (
	defaultWindowSeconds    = 60
	defaultCheckpointEvery  = 5
	defaultMaxWindowsPerRun = 100
)

// StreamingSupervisorWorkflow is a long-running Temporal workflow that:
//  1. Runs a micro-batch scan window against each streaming source.
//  2. Emits findings to the existing ingest endpoint (batch contract unchanged).
//  3. Checkpoints offset/sequence state every N windows.
//  4. Calls ContinueAsNew after maxWindowsPerRun to avoid history bloat.
//
// This preserves the batch ingestion contract: downstream services see
// ScanStatusRunning → ScanStatusCompleted per micro-batch, not a single
// unbounded run.
func StreamingSupervisorWorkflow(ctx workflow.Context, input StreamingSupervisorInput, checkpoints []StreamingCheckpointState) error {
	logger := workflow.GetLogger(ctx)

	// Apply defaults
	windowSecs := input.WindowSeconds
	if windowSecs <= 0 {
		windowSecs = defaultWindowSeconds
	}
	checkpointEvery := input.CheckpointEvery
	if checkpointEvery <= 0 {
		checkpointEvery = defaultCheckpointEvery
	}
	maxWindows := input.MaxWindowsPerRun
	if maxWindows <= 0 {
		maxWindows = defaultMaxWindowsPerRun
	}

	// Activity options for scanning activities — longer timeout for streaming windows
	scanAO := workflow.ActivityOptions{
		StartToCloseTimeout: time.Duration(windowSecs+30) * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts:    2,
			InitialInterval:    5 * time.Second,
			BackoffCoefficient: 1.5,
		},
	}

	// Activity options for ingest — must succeed, short window
	ingestAO := workflow.ActivityOptions{
		StartToCloseTimeout: 2 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts:    5,
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
		},
	}

	// Build checkpoint map keyed by "sourceType:topicOrStream"
	stateMap := make(map[string]StreamingCheckpointState)
	for _, cp := range checkpoints {
		key := cpKey(cp.Source)
		stateMap[key] = cp
	}
	// Initialize missing checkpoints
	for _, src := range input.Sources {
		key := cpKey(src)
		if _, ok := stateMap[key]; !ok {
			stateMap[key] = StreamingCheckpointState{Source: src}
		}
	}

	windowsRun := 0

	for windowsRun < maxWindows {
		windowsRun++
		logger.Info("StreamingSupervisor: starting window", "window", windowsRun)

		for _, src := range input.Sources {
			key := cpKey(src)
			state := stateMap[key]

			// Run the streaming scan window via activity
			var windowResult StreamingWindowResult
			scanCtx := workflow.WithActivityOptions(ctx, scanAO)
			err := workflow.ExecuteActivity(scanCtx, "RunStreamingWindowActivity", StreamingWindowInput{
				Source:          src,
				WindowSeconds:   windowSecs,
				ResumeFromState: state,
			}).Get(ctx, &windowResult)

			if err != nil {
				logger.Error("Streaming window failed", "source", key, "error", err)
				// Don't stop the whole supervisor — skip this window, try next
				continue
			}

			if len(windowResult.Findings) == 0 {
				logger.Info("No findings in window", "source", key)
			} else {
				// Emit findings to the standard ingest endpoint (same batch contract)
				ingestCtx := workflow.WithActivityOptions(ctx, ingestAO)
				var batchScanID string
				err = workflow.ExecuteActivity(ingestCtx, "IngestStreamingFindings", StreamingIngestInput{
					Source:      src,
					Findings:    windowResult.Findings,
					ProfileName: input.ScanProfileName,
					IngestURL:   input.IngestEndpoint,
				}).Get(ctx, &batchScanID)
				if err != nil {
					logger.Error("Failed to ingest streaming findings", "source", key, "error", err)
				} else {
					logger.Info("Streaming findings ingested",
						"source", key,
						"count", len(windowResult.Findings),
						"batchScanID", batchScanID,
					)
				}
			}

			// Update checkpoint state
			state.KafkaOffset = windowResult.LastKafkaOffset
			state.KinesisSeqNum = windowResult.LastKinesisSeqNum
			state.WindowsRun++
			state.FindingsTotal += len(windowResult.Findings)
			stateMap[key] = state
		}

		// Checkpoint every N windows
		if windowsRun%checkpointEvery == 0 {
			updatedCheckpoints := flattenCheckpoints(stateMap)
			logger.Info("Checkpointing streaming state",
				"windows", windowsRun,
				"sources", len(updatedCheckpoints),
			)
			// Persist checkpoint via activity (writes to a Redis key or DB)
			cpCtx := workflow.WithActivityOptions(ctx, ingestAO)
			_ = workflow.ExecuteActivity(cpCtx, "PersistStreamingCheckpoints", updatedCheckpoints).Get(ctx, nil)
		}
	}

	// ContinueAsNew — resets history, passes updated checkpoints forward
	updatedCheckpoints := flattenCheckpoints(stateMap)
	logger.Info("StreamingSupervisor: ContinueAsNew",
		"totalWindows", windowsRun,
		"sources", len(updatedCheckpoints),
	)
	return workflow.NewContinueAsNewError(ctx, StreamingSupervisorWorkflow, input, updatedCheckpoints)
}

// StreamingWindowInput is the input to RunStreamingWindowActivity.
type StreamingWindowInput struct {
	Source          StreamingSource          `json:"source"`
	WindowSeconds   int                      `json:"window_seconds"`
	ResumeFromState StreamingCheckpointState `json:"resume_from_state"`
}

// StreamingWindowResult is the output of RunStreamingWindowActivity.
type StreamingWindowResult struct {
	Findings          []map[string]any `json:"findings"`
	LastKafkaOffset   int64            `json:"last_kafka_offset,omitempty"`
	LastKinesisSeqNum string           `json:"last_kinesis_seq_num,omitempty"`
}

// StreamingIngestInput is the input to IngestStreamingFindings.
type StreamingIngestInput struct {
	Source      StreamingSource  `json:"source"`
	Findings    []map[string]any `json:"findings"`
	ProfileName string           `json:"profile_name"`
	IngestURL   string           `json:"ingest_url"`
}

func cpKey(src StreamingSource) string {
	return fmt.Sprintf("%s:%s", src.SourceType, src.TopicOrStream)
}

func flattenCheckpoints(m map[string]StreamingCheckpointState) []StreamingCheckpointState {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]StreamingCheckpointState, 0, len(m))
	for _, k := range keys {
		out = append(out, m[k])
	}
	return out
}
