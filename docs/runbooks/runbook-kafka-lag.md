# Runbook: Kafka Consumer Lag

## Trigger

The ARC-Hawk scanner's Kafka consumer group falls behind the latest offset on one or more topic partitions. Consumer lag grows when the scanner cannot process messages as fast as they are produced, or when the scanner is offline and messages accumulate.

Activates when consumer lag on any monitored topic exceeds the alert threshold (default: 10,000 messages) or when the scanner logs `WARN kafka_consumer lag_growing`.

---

## Symptoms

**Logs (scanner worker):**
```
WARN   hawk_scanner  source=kafka  topic=raw-events  partition=2  consumer_group=hawk-scanner
         lag=54321  last_committed_offset=100000  latest_offset=154321  lag_growing=true
```

**Kafka metrics (via Kafka Exporter or Confluent Control Center):**
```
kafka_consumergroup_lag{topic="raw-events", partition="2", group="hawk-scanner"} 54321
```

**Prometheus alert:** `KafkaScannerLagHigh` fires when `kafka_consumergroup_lag > 10000` for more than 5 minutes.

**Dashboard:** Connections page shows Kafka connection with `health_status: lagging` and a lag count badge.

---

## Automated Response

The scanner monitors its own consumer lag after each poll cycle. When lag exceeds the configured warning threshold (`kafka.lag_warn_threshold`, default 10,000):

1. It logs a warning with the topic, partition, and lag count.
2. It sets `connection.health_status = 'lagging'` on the Kafka connection record.
3. It does **not** self-remediate — lag clearing requires operator action or organic catch-up.

When lag exceeds the critical threshold (`kafka.lag_critical_threshold`, default 100,000):

1. An alert is sent via the configured notification channel (email/Slack/PagerDuty).
2. The scanner increases its batch poll size to `kafka.max_poll_records_high_lag` (default 2000, up from 500) to accelerate catch-up.

---

## Manual Steps

### Step 1: Diagnose the cause

```bash
# Check current lag for the scanner consumer group
kafka-consumer-groups.sh --bootstrap-server localhost:9092 \
  --describe --group hawk-scanner

# Check if any partitions are leader-less (broker issues)
kafka-topics.sh --bootstrap-server localhost:9092 \
  --describe --topic raw-events | grep "Leader: -1"

# Check scanner process health
curl http://localhost:9090/health  # agent health (if via agent)
```

Identify whether the lag is:
- **Organic lag** — scanner is running but producing traffic is faster than consumption.
- **Scanner down** — scanner process crashed; lag accumulated while offline.
- **Broker issue** — a Kafka broker is down causing partition leadership gaps.

### Step 2: If the scanner is down

```bash
# Restart the scanner worker (Docker)
docker-compose restart scanner

# Or via systemd
systemctl restart hawk-scanner

# Verify it reconnects and consumption resumes
docker-compose logs -f scanner | grep kafka
```

### Step 3: If the scanner is running but lag is growing (throughput issue)

1. **Increase consumer parallelism** — add more partitions to the topic (requires broker admin):
   ```bash
   kafka-topics.sh --bootstrap-server localhost:9092 \
     --alter --topic raw-events --partitions 16
   ```

2. **Scale out scanner workers** — run additional scanner instances (each will consume from a subset of partitions):
   ```bash
   docker-compose up --scale scanner=3 -d
   ```

3. **Temporarily reduce scan complexity** — disable expensive validators (e.g., OCR, Parquet deep scan) until lag clears:
   ```yaml
   # hawk_scanner/config.yml
   validators:
     enable_ocr: false  # re-enable after catch-up
   ```

### Step 4: If a Kafka broker is down

```bash
# Identify the broker
kafka-broker-api-versions.sh --bootstrap-server broker1:9092,broker2:9092,broker3:9092

# Restart the broker (Docker)
docker-compose restart kafka

# Or in Kubernetes
kubectl rollout restart deployment/kafka -n messaging
```

Wait for partition leader election to complete (typically <30 seconds), then verify lag resumes draining.

### Step 5: Reset consumer offset (last resort)

Only do this if you accept that messages will be reprocessed (and possibly produce duplicate findings — handled by deduplication in the ingestion pipeline):

```bash
# Reset to the latest offset (skip all historical lag — use only for topic resets)
kafka-consumer-groups.sh --bootstrap-server localhost:9092 \
  --group hawk-scanner --topic raw-events \
  --reset-offsets --to-latest --execute

# Or reset to a specific timestamp (safer — reprocess only the last 2 hours)
kafka-consumer-groups.sh --bootstrap-server localhost:9092 \
  --group hawk-scanner --topic raw-events \
  --reset-offsets --to-datetime 2026-04-09T10:00:00.000 --execute
```

**Warning:** Resetting to `--to-latest` permanently skips all messages in the lag. The findings from those messages will not be generated. Only do this if you can accept the data gap.

---

## Resolution Criteria

- `kafka_consumergroup_lag` for `hawk-scanner` returns to < 1,000 on all partitions.
- The `KafkaScannerLagHigh` Prometheus alert resolves.
- The Kafka connection in the dashboard shows `health_status: ok`.
- No `lag_growing=true` in scanner logs for at least 5 consecutive minutes.

---

## Prevention

- **Right-size partitions at topic creation:** Provision at least as many partitions as the expected number of scanner worker instances. More partitions allow more parallel consumers.
- **Capacity planning:** Monitor `kafka_consumergroup_lag` as a leading indicator. If lag consistently grows during peak hours, increase scanner throughput or add partitions before it reaches the critical threshold.
- **Message retention policy:** Set a retention period on the Kafka topic (`retention.ms`) appropriate for your scanner's expected downtime window. If the scanner can be offline for 8 hours, set retention to at least 24 hours to allow catch-up.
- **Lag alerting:** Configure a Prometheus/Alertmanager alert on `kafka_consumergroup_lag > 10000` for more than 5 minutes so on-call is notified before the lag becomes critical.
- **Deduplication in ingestion:** The finding ingestion pipeline should be idempotent (deduplicate on `(scan_job_id, batch_seq, source_hash)`) so that offset resets or replays do not create duplicate findings.
