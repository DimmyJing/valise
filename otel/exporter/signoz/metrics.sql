CREATE DATABASE IF NOT EXISTS signoz_metrics;

-- samples_v2

CREATE TABLE IF NOT EXISTS signoz_metrics.samples_v2
(
    `metric_name` LowCardinality(String),
    `fingerprint` UInt64 CODEC(DoubleDelta, LZ4),
    `timestamp_ms` Int64 CODEC(DoubleDelta, LZ4),
    `value` Float64 CODEC(Gorilla, LZ4)
)
ENGINE = MergeTree
PARTITION BY toDate(timestamp_ms / 1000)
ORDER BY (metric_name, fingerprint, timestamp_ms)
TTL toDateTime(timestamp_ms / 1000) + toIntervalSecond(2592000)
SETTINGS index_granularity = 8192, ttl_only_drop_parts = 1;

-- time_series_v2

CREATE TABLE IF NOT EXISTS signoz_metrics.time_series_v2
(
    `metric_name` LowCardinality(String),
    `fingerprint` UInt64 CODEC(DoubleDelta, LZ4),
    `timestamp_ms` Int64 CODEC(DoubleDelta, LZ4),
    `labels` String CODEC(ZSTD(5)),
    `temporality` LowCardinality(String) DEFAULT 'Unspecified' CODEC(ZSTD(5)),
    INDEX temporality_index temporality TYPE SET(3) GRANULARITY 1
)
ENGINE = ReplacingMergeTree
PARTITION BY toDate(timestamp_ms / 1000)
ORDER BY (metric_name, fingerprint)
SETTINGS index_granularity = 8192, ttl_only_drop_parts = 1;

-- time_series_v3

CREATE TABLE IF NOT EXISTS signoz_metrics.time_series_v3
(
    `env` LowCardinality(String) DEFAULT 'default',
    `temporality` LowCardinality(String) DEFAULT 'Unspecified',
    `metric_name` LowCardinality(String),
    `fingerprint` UInt64 CODEC(Delta(8), ZSTD(1)),
    `timestamp_ms` Int64 CODEC(Delta(8), ZSTD(1)),
    `labels` String CODEC(ZSTD(5))
)
ENGINE = ReplacingMergeTree
PARTITION BY toDate(timestamp_ms / 1000)
ORDER BY (env, temporality, metric_name, fingerprint)
SETTINGS index_granularity = 8192;

-- usage

CREATE TABLE IF NOT EXISTS signoz_metrics.usage
(
    `tenant` String,
    `collector_id` String,
    `exporter_id` String,
    `timestamp` DateTime,
    `data` String
)
ENGINE = MergeTree
ORDER BY (tenant, collector_id, exporter_id, timestamp)
TTL timestamp + toIntervalDay(3)
SETTINGS index_granularity = 8192;
