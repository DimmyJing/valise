package signoz

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
)

func formatKey(k string) string {
	return strings.ReplaceAll(k, ".", "_")
}

func attributesToSlice(attributes pcommon.Map, forceStringValues bool) attributesToSliceResponse {
	var response attributesToSliceResponse

	attributes.Range(func(key string, value pcommon.Value) bool {
		if forceStringValues {
			// store everything as string
			response.StringKeys = append(response.StringKeys, formatKey(key))
			response.StringValues = append(response.StringValues, value.AsString())
		} else {
			//nolint:exhaustive
			switch value.Type() {
			case pcommon.ValueTypeInt:
				response.IntKeys = append(response.IntKeys, formatKey(key))
				response.IntValues = append(response.IntValues, value.Int())
			case pcommon.ValueTypeDouble:
				response.FloatKeys = append(response.FloatKeys, formatKey(key))
				response.FloatValues = append(response.FloatValues, value.Double())
			case pcommon.ValueTypeBool:
				response.BoolKeys = append(response.BoolKeys, formatKey(key))
				response.BoolValues = append(response.BoolValues, value.Bool())
			default: // store it as string
				response.StringKeys = append(response.StringKeys, formatKey(key))
				response.StringValues = append(response.StringValues, value.AsString())
			}
		}

		return true
	})

	return response
}

type attributesToSliceResponse struct {
	StringKeys   []string
	StringValues []string
	IntKeys      []string
	IntValues    []int64
	FloatKeys    []string
	FloatValues  []float64
	BoolKeys     []string
	BoolValues   []bool
}

func addAttrsToTagStatement(
	statement driver.Batch,
	tagType string,
	attrs attributesToSliceResponse,
) error {
	for idx, v := range attrs.StringKeys {
		if err := statement.Append(time.Now(), v, tagType, "string", attrs.StringValues[idx], nil, nil); err != nil {
			return fmt.Errorf("could not append string attribute to batch: %w", err)
		}
	}

	for idx, v := range attrs.IntKeys {
		if err := statement.Append(time.Now(), v, tagType, "int64", nil, attrs.IntValues[idx], nil); err != nil {
			return fmt.Errorf("could not append number attribute to batch: %w", err)
		}
	}

	for idx, v := range attrs.FloatKeys {
		if err := statement.Append(time.Now(), v, tagType, "float64", nil, nil, attrs.FloatValues[idx]); err != nil {
			return fmt.Errorf("could not append number attribute to batch: %w", err)
		}
	}

	for _, v := range attrs.BoolKeys {
		if err := statement.Append(time.Now(), v, tagType, "bool", nil, nil, nil); err != nil {
			return fmt.Errorf("could not append bool attribute to batch: %w", err)
		}
	}

	return nil
}

func getStringifiedBody(body pcommon.Value) string {
	if body.Type() == pcommon.ValueTypeBytes {
		return string(body.Bytes().AsRaw())
	}

	return body.AsString()
}

func (s *signozExporter) convertLogs(ctx context.Context, logs plog.Logs) error { //nolint:funlen,cyclop
	statement, err := s.conn.PrepareBatch(ctx, fmt.Sprintf("INSERT INTO %s.%s", logsDB, logsTable))
	if err != nil {
		return fmt.Errorf("error preparing logs table statement: %w", err)
	}

	tagStatement, err := s.conn.PrepareBatch(ctx, fmt.Sprintf("INSERT INTO %s.%s", logsDB, tagAttributeTable))
	if err != nil {
		return fmt.Errorf("error preparing tagAttribute table statement: %w", err)
	}

	for i := 0; i < logs.ResourceLogs().Len(); i++ {
		resourceLogs := logs.ResourceLogs().At(i)
		resource := resourceLogs.Resource()
		resources := attributesToSlice(resource.Attributes(), true)

		err = addAttrsToTagStatement(tagStatement, "resource", resources)
		if err != nil {
			return fmt.Errorf("error adding resource attributes to tag statement: %w", err)
		}

		for j := 0; j < resourceLogs.ScopeLogs().Len(); j++ {
			logRecords := resourceLogs.ScopeLogs().At(j).LogRecords()
			for k := 0; k < logRecords.Len(); k++ {
				logRecord := logRecords.At(k)

				timestamp := uint64(logRecord.Timestamp())
				observedTimestamp := uint64(logRecord.ObservedTimestamp())

				if observedTimestamp == 0 {
					observedTimestamp = uint64(time.Now().UnixNano())
				}

				if timestamp == 0 {
					timestamp = observedTimestamp
				}

				attributes := attributesToSlice(logRecord.Attributes(), false)

				err := addAttrsToTagStatement(tagStatement, "tag", attributes)
				if err != nil {
					return err
				}

				err = statement.Append(
					timestamp,
					observedTimestamp,
					s.ksuid.String(),
					TraceIDToHexOrEmptyString(logRecord.TraceID()),
					SpanIDToHexOrEmptyString(logRecord.SpanID()),
					uint32(logRecord.Flags()),
					logRecord.SeverityText(),
					uint8(logRecord.SeverityNumber()),
					getStringifiedBody(logRecord.Body()),
					resources.StringKeys,
					resources.StringValues,
					attributes.StringKeys,
					attributes.StringValues,
					attributes.IntKeys,
					attributes.IntValues,
					attributes.FloatKeys,
					attributes.FloatValues,
					attributes.BoolKeys,
					attributes.BoolValues,
				)
				if err != nil {
					return fmt.Errorf("failed to append statement to logs: %w", err)
				}

				s.ksuid = s.ksuid.Next()
			}
		}
	}

	err = statement.Send()
	if err != nil {
		return fmt.Errorf("failed to send logs to clickhouse: %w", err)
	}

	err = tagStatement.Send()
	if err != nil {
		return fmt.Errorf("failed to send tags to clickhouse: %w", err)
	}

	return nil
}
