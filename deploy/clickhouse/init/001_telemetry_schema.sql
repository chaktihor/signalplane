CREATE DATABASE IF NOT EXISTS signalplane;

CREATE TABLE IF NOT EXISTS signalplane.metrics
(
  timestamp DateTime64(9, 'UTC'),
  organization_id LowCardinality(String),
  environment LowCardinality(String),
  service LowCardinality(String),
  host LowCardinality(String),
  region LowCardinality(String),
  metric_name LowCardinality(String),
  metric_type LowCardinality(String),
  unit LowCardinality(String),
  value Float64,
  labels Map(String, String),
  resource_attributes Map(String, String),
  ingest_time DateTime64(9, 'UTC') DEFAULT now64(9)
)
ENGINE = MergeTree
PARTITION BY toDate(timestamp)
ORDER BY (organization_id, environment, metric_name, service, host, timestamp)
TTL toDateTime(timestamp) + INTERVAL 30 DAY;

CREATE TABLE IF NOT EXISTS signalplane.logs
(
  timestamp DateTime64(9, 'UTC'),
  organization_id LowCardinality(String),
  environment LowCardinality(String),
  service LowCardinality(String),
  host LowCardinality(String),
  region LowCardinality(String),
  severity LowCardinality(String),
  message String,
  trace_id String,
  span_id String,
  fields Map(String, String),
  resource_attributes Map(String, String),
  ingest_time DateTime64(9, 'UTC') DEFAULT now64(9)
)
ENGINE = MergeTree
PARTITION BY toDate(timestamp)
ORDER BY (organization_id, environment, service, severity, timestamp)
TTL toDateTime(timestamp) + INTERVAL 30 DAY;

CREATE TABLE IF NOT EXISTS signalplane.traces
(
  timestamp DateTime64(9, 'UTC'),
  organization_id LowCardinality(String),
  environment LowCardinality(String),
  trace_id String,
  root_service LowCardinality(String),
  operation String,
  duration_ms Float64,
  status LowCardinality(String),
  resource_attributes Map(String, String),
  ingest_time DateTime64(9, 'UTC') DEFAULT now64(9)
)
ENGINE = MergeTree
PARTITION BY toDate(timestamp)
ORDER BY (organization_id, environment, root_service, status, timestamp)
TTL toDateTime(timestamp) + INTERVAL 30 DAY;

CREATE TABLE IF NOT EXISTS signalplane.spans
(
  timestamp DateTime64(9, 'UTC'),
  organization_id LowCardinality(String),
  environment LowCardinality(String),
  trace_id String,
  span_id String,
  parent_span_id String,
  service LowCardinality(String),
  name String,
  duration_ms Float64,
  status LowCardinality(String),
  attributes Map(String, String),
  ingest_time DateTime64(9, 'UTC') DEFAULT now64(9)
)
ENGINE = MergeTree
PARTITION BY toDate(timestamp)
ORDER BY (organization_id, environment, trace_id, span_id)
TTL toDateTime(timestamp) + INTERVAL 30 DAY;

CREATE TABLE IF NOT EXISTS signalplane.uptime_results
(
  timestamp DateTime64(9, 'UTC'),
  organization_id LowCardinality(String),
  monitor_id String,
  name String,
  url String,
  status LowCardinality(String),
  expected_status UInt16,
  status_code UInt16,
  response_ms Float64,
  error String,
  ingest_time DateTime64(9, 'UTC') DEFAULT now64(9)
)
ENGINE = MergeTree
PARTITION BY toDate(timestamp)
ORDER BY (organization_id, monitor_id, timestamp)
TTL toDateTime(timestamp) + INTERVAL 30 DAY;

CREATE TABLE IF NOT EXISTS signalplane.events
(
  timestamp DateTime64(9, 'UTC'),
  organization_id LowCardinality(String),
  environment LowCardinality(String),
  source LowCardinality(String),
  event_type LowCardinality(String),
  entity_id String,
  severity LowCardinality(String),
  message String,
  attributes Map(String, String),
  ingest_time DateTime64(9, 'UTC') DEFAULT now64(9)
)
ENGINE = MergeTree
PARTITION BY toDate(timestamp)
ORDER BY (organization_id, environment, source, event_type, timestamp)
TTL toDateTime(timestamp) + INTERVAL 30 DAY;
