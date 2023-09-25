package transform

import (
	"github.com/DimmyJing/valise/otel/otellog"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/otel/attribute"
)

func Logs(logs []otellog.ReadOnlyLog) plog.Logs {
	pLogs := plog.NewLogs()
	if len(logs) == 0 {
		return pLogs
	}

	resourceLogsMap := make(map[attribute.Distinct]plog.ResourceLogs)
	scopeLogsMap := make(map[scopeKey]plog.ScopeLogs)
	resourceLogs := pLogs.ResourceLogs()

	for _, logData := range logs {
		if logData == nil {
			continue
		}

		resourceKey := logData.Resource().Equivalent()
		scopeKey := scopeKey{
			r:  resourceKey,
			is: logData.InstrumentationScope(),
		}

		resourceLog, rOk := resourceLogsMap[resourceKey]
		if !rOk {
			// The resource was unknown.
			resourceLog = resourceLogs.AppendEmpty()
			logResource := logData.Resource()
			transformResource(logResource, resourceLog.Resource())
			resourceLog.SetSchemaUrl(logResource.SchemaURL())
			resourceLogsMap[resourceKey] = resourceLog
		}

		scopeLog, iOk := scopeLogsMap[scopeKey]
		if !iOk {
			// Either the resource or instrumentation scope were unknown.
			scopeLog = resourceLog.ScopeLogs().AppendEmpty()
			instrumentationScope := logData.InstrumentationScope()
			transformInstrumentationScope(instrumentationScope, scopeLog.Scope())
			scopeLog.SetSchemaUrl(instrumentationScope.SchemaURL)
			scopeLogsMap[scopeKey] = scopeLog
		}

		transformLog(logData, scopeLog.LogRecords().AppendEmpty())
	}

	return pLogs
}

func transformLog(logData otellog.ReadOnlyLog, pLog plog.LogRecord) {
	pLog.SetObservedTimestamp(pcommon.NewTimestampFromTime(logData.ObservedTime()))
	pLog.SetTimestamp(pcommon.NewTimestampFromTime(logData.Time()))
	pLog.SetTraceID(logData.TraceID())
	pLog.SetSpanID(logData.SpanID())
	pLog.SetFlags(logFlags(logData.Flags()))
	pLog.SetSeverityText(logData.SeverityText())
	pLog.SetSeverityNumber(severityNumber(logData.SeverityNumber()))
	transformValue(logData.Body(), pLog.Body())
	TransformKeyValues(logData.Attributes(), pLog.Attributes())
	pLog.SetDroppedAttributesCount(logData.DroppedAttributesCount())
}

func logFlags(flags otellog.LogFlags) plog.LogRecordFlags {
	switch flags {
	case otellog.LogFlagsDefault:
		return plog.DefaultLogRecordFlags
	case otellog.LogFlagsIsSampled:
		return plog.DefaultLogRecordFlags.WithIsSampled(true)
	default:
		return plog.DefaultLogRecordFlags
	}
}

func severityNumber(severity otellog.SeverityNumber) plog.SeverityNumber {
	switch severity {
	case otellog.SeverityNumberTrace:
		return plog.SeverityNumberTrace
	case otellog.SeverityNumberDebug:
		return plog.SeverityNumberDebug
	case otellog.SeverityNumberInfo:
		return plog.SeverityNumberInfo
	case otellog.SeverityNumberWarn:
		return plog.SeverityNumberWarn
	case otellog.SeverityNumberError:
		return plog.SeverityNumberError
	case otellog.SeverityNumberFatal:
		return plog.SeverityNumberFatal
	default:
		return plog.SeverityNumberUnspecified
	}
}
