CREATE DATABASE IF NOT EXISTS signoz_logs;

-- logs

CREATE TABLE IF NOT EXISTS signoz_logs.logs
(
    `timestamp` UInt64 CODEC(DoubleDelta, LZ4),
    `observed_timestamp` UInt64 CODEC(DoubleDelta, LZ4),
    `id` String CODEC(ZSTD(1)),
    `trace_id` String CODEC(ZSTD(1)),
    `span_id` String CODEC(ZSTD(1)),
    `trace_flags` UInt32,
    `severity_text` LowCardinality(String) CODEC(ZSTD(1)),
    `severity_number` UInt8,
    `body` String CODEC(ZSTD(2)),
    `resources_string_key` Array(String) CODEC(ZSTD(1)),
    `resources_string_value` Array(String) CODEC(ZSTD(1)),
    `attributes_string_key` Array(String) CODEC(ZSTD(1)),
    `attributes_string_value` Array(String) CODEC(ZSTD(1)),
    `attributes_int64_key` Array(String) CODEC(ZSTD(1)),
    `attributes_int64_value` Array(Int64) CODEC(ZSTD(1)),
    `attributes_float64_key` Array(String) CODEC(ZSTD(1)),
    `attributes_float64_value` Array(Float64) CODEC(ZSTD(1)),
    `attributes_bool_key` Array(String) CODEC(ZSTD(1)),
    `attributes_bool_value` Array(Bool) CODEC(ZSTD(1)),
    INDEX body_idx body TYPE tokenbf_v1(10240, 3, 0) GRANULARITY 4,
    INDEX id_minmax id TYPE minmax GRANULARITY 1,
    INDEX severity_number_idx severity_number TYPE set(25) GRANULARITY 4,
    INDEX severity_text_idx severity_text TYPE set(25) GRANULARITY 4,
    INDEX trace_flags_idx trace_flags TYPE bloom_filter GRANULARITY 4
)
ENGINE = MergeTree
PARTITION BY toDate(timestamp / 1000000000)
ORDER BY (timestamp, id)
TTL toDateTime(timestamp / 1000000000) + toIntervalSecond(604800)
SETTINGS index_granularity = 8192, ttl_only_drop_parts = 1;

-- logs_attribute_keys

CREATE TABLE IF NOT EXISTS signoz_logs.logs_attribute_keys
(
    `name` String,
    `datatype` String
)
ENGINE = ReplacingMergeTree
ORDER BY (name, datatype)
SETTINGS index_granularity = 8192;

-- attribute_keys_float64_final_mv

CREATE MATERIALIZED VIEW IF NOT EXISTS signoz_logs.attribute_keys_float64_final_mv TO signoz_logs.logs_attribute_keys
(
    `name` String,
    `datatype` String
) AS
SELECT DISTINCT
    arrayJoin(attributes_float64_key) AS name,
    'Float64' AS datatype
FROM signoz_logs.logs
ORDER BY name ASC;

-- attribute_keys_int64_final_mv

CREATE MATERIALIZED VIEW IF NOT EXISTS signoz_logs.attribute_keys_int64_final_mv TO signoz_logs.logs_attribute_keys
(
    `name` String,
    `datatype` String
) AS
SELECT DISTINCT
    arrayJoin(attributes_int64_key) AS name,
    'Int64' AS datatype
FROM signoz_logs.logs
ORDER BY name ASC;

-- attribute_keys_string_final_mv

CREATE MATERIALIZED VIEW IF NOT EXISTS signoz_logs.attribute_keys_string_final_mv TO signoz_logs.logs_attribute_keys
(
    `name` String,
    `datatype` String
) AS
SELECT DISTINCT
    arrayJoin(attributes_string_key) AS name,
    'String' AS datatype
FROM signoz_logs.logs
ORDER BY name ASC;

-- attribute_keys_bool_final_mv

CREATE MATERIALIZED VIEW IF NOT EXISTS signoz_logs.attribute_keys_bool_final_mv TO signoz_logs.logs_attribute_keys
(
    `name` String,
    `datatype` String
) AS
SELECT DISTINCT
    arrayJoin(attributes_bool_key) AS name,
    'Bool' AS datatype
FROM signoz_logs.logs
ORDER BY name ASC;

-- logs_resource_keys

CREATE TABLE IF NOT EXISTS signoz_logs.logs_resource_keys
(
    `name` String,
    `datatype` String
)
ENGINE = ReplacingMergeTree
ORDER BY (name, datatype)
SETTINGS index_granularity = 8192;

-- resource_keys_string_final_mv

CREATE MATERIALIZED VIEW IF NOT EXISTS signoz_logs.resource_keys_string_final_mv TO signoz_logs.logs_resource_keys
(
    `name` String,
    `datatype` String
) AS
SELECT DISTINCT
    arrayJoin(resources_string_key) AS name,
    'String' AS datatype
FROM signoz_logs.logs
ORDER BY name ASC;

-- tag_attributes

CREATE TABLE IF NOT EXISTS signoz_logs.tag_attributes
(
    `timestamp` DateTime CODEC(ZSTD(1)),
    `tagKey` LowCardinality(String) CODEC(ZSTD(1)),
    `tagType` Enum8('tag' = 1, 'resource' = 2) CODEC(ZSTD(1)),
    `tagDataType` Enum8('string' = 1, 'bool' = 2, 'int64' = 3, 'float64' = 4) CODEC(ZSTD(1)),
    `stringTagValue` String CODEC(ZSTD(1)),
    `int64TagValue` Nullable(Int64) CODEC(ZSTD(1)),
    `float64TagValue` Nullable(Float64) CODEC(ZSTD(1))
)
ENGINE = ReplacingMergeTree
ORDER BY (tagKey, tagType, tagDataType, stringTagValue, int64TagValue, float64TagValue)
TTL toDateTime(timestamp) + toIntervalSecond(172800)
SETTINGS ttl_only_drop_parts = 1, allow_nullable_key = 1, index_granularity = 8192;

-- usage

CREATE TABLE IF NOT EXISTS signoz_logs.usage
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
