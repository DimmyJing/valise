package signoz

import (
	"context"
	"crypto/md5" //nolint:gosec
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

type Event struct {
	Name         string            `json:"name,omitempty"`
	TimeUnixNano uint64            `json:"timeUnixNano,omitempty"`
	AttributeMap map[string]string `json:"attributeMap,omitempty"`
	IsError      bool              `json:"isError,omitempty"`
}

type OtelSpanRef struct {
	TraceID string `json:"traceId,omitempty"`
	SpanID  string `json:"spanId,omitempty"`
	RefType string `json:"refType,omitempty"`
}

type references []OtelSpanRef

type TraceModel struct {
	TraceID           string             `json:"traceId,omitempty"`
	SpanID            string             `json:"spanId,omitempty"`
	Name              string             `json:"name,omitempty"`
	DurationNano      uint64             `json:"durationNano,omitempty"`
	StartTimeUnixNano uint64             `json:"startTimeUnixNano,omitempty"`
	ServiceName       string             `json:"serviceName,omitempty"`
	Kind              int8               `json:"kind,omitempty"`
	References        references         `json:"references,omitempty"`
	StatusCode        int16              `json:"statusCode,omitempty"`
	TagMap            map[string]string  `json:"tagMap,omitempty"`
	StringTagMap      map[string]string  `json:"stringTagMap,omitempty"`
	NumberTagMap      map[string]float64 `json:"numberTagMap,omitempty"`
	BoolTagMap        map[string]bool    `json:"boolTagMap,omitempty"`
	Events            []string           `json:"event,omitempty"`
	HasError          bool               `json:"hasError,omitempty"`
}

type SpanAttribute struct {
	Key         string
	TagType     string
	DataType    string
	StringValue string
	NumberValue float64
	IsColumn    bool
}

func (s SpanAttribute) ToKey() SpanAttributeKey {
	return SpanAttributeKey{
		Key:      s.Key,
		TagType:  s.TagType,
		DataType: s.DataType,
		IsColumn: s.IsColumn,
	}
}

type SpanAttributeKey struct {
	Key      string
	TagType  string
	DataType string
	IsColumn bool
}

//nolint:tagliatelle
type Span struct {
	TraceID            string             `json:"traceId,omitempty"`
	SpanID             string             `json:"spanId,omitempty"`
	ParentSpanID       string             `json:"parentSpanId,omitempty"`
	Name               string             `json:"name,omitempty"`
	DurationNano       uint64             `json:"durationNano,omitempty"`
	StartTimeUnixNano  uint64             `json:"startTimeUnixNano,omitempty"`
	ServiceName        string             `json:"serviceName,omitempty"`
	Kind               int8               `json:"kind,omitempty"`
	StatusCode         int16              `json:"statusCode,omitempty"`
	ExternalHTTPMethod string             `json:"externalHttpMethod,omitempty"`
	HTTPURL            string             `json:"httpUrl,omitempty"`
	HTTPMethod         string             `json:"httpMethod,omitempty"`
	HTTPHost           string             `json:"httpHost,omitempty"`
	HTTPRoute          string             `json:"httpRoute,omitempty"`
	HTTPCode           string             `json:"httpCode,omitempty"`
	MsgSystem          string             `json:"msgSystem,omitempty"`
	MsgOperation       string             `json:"msgOperation,omitempty"`
	ExternalHTTPURL    string             `json:"externalHttpUrl,omitempty"`
	Component          string             `json:"component,omitempty"`
	DBSystem           string             `json:"dbSystem,omitempty"`
	DBName             string             `json:"dbName,omitempty"`
	DBOperation        string             `json:"dbOperation,omitempty"`
	PeerService        string             `json:"peerService,omitempty"`
	Events             []string           `json:"event,omitempty"`
	ErrorEvent         Event              `json:"errorEvent,omitempty"`
	ErrorID            string             `json:"errorID,omitempty"`
	ErrorGroupID       string             `json:"errorGroupID,omitempty"`
	TagMap             map[string]string  `json:"tagMap,omitempty"`
	StringTagMap       map[string]string  `json:"stringTagMap,omitempty"`
	NumberTagMap       map[string]float64 `json:"numberTagMap,omitempty"`
	BoolTagMap         map[string]bool    `json:"boolTagMap,omitempty"`
	ResourceTagsMap    map[string]string  `json:"resourceTagsMap,omitempty"`
	HasError           bool               `json:"hasError,omitempty"`
	TraceModel         TraceModel         `json:"traceModel,omitempty"`
	GRPCCode           string             `json:"gRPCCode,omitempty"`
	GRPCMethod         string             `json:"gRPCMethod,omitempty"`
	RPCSystem          string             `json:"rpcSystem,omitempty"`
	RPCService         string             `json:"rpcService,omitempty"`
	RPCMethod          string             `json:"rpcMethod,omitempty"`
	ResponseStatusCode string             `json:"responseStatusCode,omitempty"`
	Tenant             *string            `json:"-"`
	SpanAttributes     []SpanAttribute    `json:"spanAttributes,omitempty"`
}

func TraceIDToHexOrEmptyString(traceID pcommon.TraceID) string {
	if !traceID.IsEmpty() {
		return hex.EncodeToString(traceID[:])
	}

	return ""
}

func SpanIDToHexOrEmptyString(spanID pcommon.SpanID) string {
	if !spanID.IsEmpty() {
		return hex.EncodeToString(spanID[:])
	}

	return ""
}

func makeJaegerProtoReferences(
	links ptrace.SpanLinkSlice,
	parentSpanID pcommon.SpanID,
	traceID pcommon.TraceID,
) []OtelSpanRef {
	parentSpanIDSet := len([8]byte(parentSpanID)) != 0
	if !parentSpanIDSet && links.Len() == 0 {
		return nil
	}

	refsCount := links.Len()
	if parentSpanIDSet {
		refsCount++
	}

	refs := make([]OtelSpanRef, 0, refsCount)

	if parentSpanIDSet {
		refs = append(refs, OtelSpanRef{
			TraceID: TraceIDToHexOrEmptyString(traceID),
			SpanID:  SpanIDToHexOrEmptyString(parentSpanID),
			RefType: "CHILD_OF",
		})
	}

	for i := 0; i < links.Len(); i++ {
		link := links.At(i)
		refs = append(refs, OtelSpanRef{
			TraceID: TraceIDToHexOrEmptyString(link.TraceID()),
			SpanID:  SpanIDToHexOrEmptyString(link.SpanID()),
			RefType: "FOLLOWS_FROM",
		})
	}

	return refs
}

func addExtraAttr(
	attributes []SpanAttribute,
	key string,
	dataType string,
	stringValue string,
	numberValue float64,
) []SpanAttribute {
	return append(attributes, SpanAttribute{
		Key:         key,
		TagType:     "tag",
		IsColumn:    true,
		DataType:    dataType,
		StringValue: stringValue,
		NumberValue: numberValue,
	})
}

func extractSpanAttributesFromSpanIndex(span *Span) []SpanAttribute {
	attrs := []SpanAttribute{}
	attrs = addExtraAttr(attrs, "traceID", "string", span.TraceID, 0)
	attrs = addExtraAttr(attrs, "spanID", "string", span.SpanID, 0)
	attrs = addExtraAttr(attrs, "parentSpanID", "string", span.ParentSpanID, 0)
	attrs = addExtraAttr(attrs, "name", "string", span.Name, 0)
	attrs = addExtraAttr(attrs, "serviceName", "string", span.ServiceName, 0)
	attrs = addExtraAttr(attrs, "kind", "float64", "", float64(span.Kind))
	attrs = addExtraAttr(attrs, "durationNano", "float64", "", float64(span.DurationNano))
	attrs = addExtraAttr(attrs, "statusCode", "float64", "", float64(span.StatusCode))
	attrs = addExtraAttr(attrs, "hasError", "bool", "", 0)
	attrs = addExtraAttr(attrs, "externalHttpMethod", "string", span.ExternalHTTPMethod, 0)
	attrs = addExtraAttr(attrs, "externalHttpUrl", "string", span.ExternalHTTPURL, 0)
	attrs = addExtraAttr(attrs, "component", "string", span.Component, 0)
	attrs = addExtraAttr(attrs, "dbSystem", "string", span.DBSystem, 0)
	attrs = addExtraAttr(attrs, "dbName", "string", span.DBName, 0)
	attrs = addExtraAttr(attrs, "dbOperation", "string", span.DBOperation, 0)
	attrs = addExtraAttr(attrs, "peerService", "string", span.PeerService, 0)
	attrs = addExtraAttr(attrs, "httpMethod", "string", span.HTTPMethod, 0)
	attrs = addExtraAttr(attrs, "httpUrl", "string", span.HTTPURL, 0)
	attrs = addExtraAttr(attrs, "httpRoute", "string", span.HTTPRoute, 0)
	attrs = addExtraAttr(attrs, "httpHost", "string", span.HTTPHost, 0)
	attrs = addExtraAttr(attrs, "msgSystem", "string", span.MsgSystem, 0)
	attrs = addExtraAttr(attrs, "msgOperation", "string", span.MsgOperation, 0)
	attrs = addExtraAttr(attrs, "rpcSystem", "string", span.RPCSystem, 0)
	attrs = addExtraAttr(attrs, "rpcService", "string", span.RPCService, 0)
	attrs = addExtraAttr(attrs, "rpcMethod", "string", span.RPCMethod, 0)
	attrs = addExtraAttr(attrs, "responseStatusCode", "string", span.ResponseStatusCode, 0)

	return attrs
}

func (s *signozExporter) convertSpan( //nolint:funlen,gocognit,gocyclo,cyclop,maintidx
	otelSpan ptrace.Span,
	serviceName string,
	resource pcommon.Resource,
) *Span {
	durationNano := uint64(otelSpan.EndTimestamp() - otelSpan.StartTimestamp())
	attributes := otelSpan.Attributes()
	resourceAttributes := resource.Attributes()
	tagMap := make(map[string]string)
	stringTagMap := make(map[string]string)
	numberTagMap := make(map[string]float64)
	boolTagMap := make(map[string]bool)
	spanAttributes := []SpanAttribute{}
	resourceAttrs := make(map[string]string)

	attributes.Range(func(key string, value pcommon.Value) bool {
		tagMap[key] = value.AsString()
		spanAttribute := SpanAttribute{
			Key:         key,
			TagType:     "tag",
			IsColumn:    false,
			DataType:    "",
			StringValue: "",
			NumberValue: 0,
		}
		//nolint:exhaustive
		switch value.Type() {
		case pcommon.ValueTypeDouble:
			numberTagMap[key] = value.Double()
			spanAttribute.NumberValue = value.Double()
			//nolint:goconst
			spanAttribute.DataType = "float64"
		case pcommon.ValueTypeInt:
			numberTagMap[key] = float64(value.Int())
			spanAttribute.NumberValue = float64(value.Int())
			spanAttribute.DataType = "float64"
		case pcommon.ValueTypeBool:
			boolTagMap[key] = value.Bool()
			//nolint:goconst
			spanAttribute.DataType = "bool"
		default:
			stringTagMap[key] = value.AsString()
			spanAttribute.StringValue = value.AsString()
			//nolint:goconst
			spanAttribute.DataType = "string"
		}
		spanAttributes = append(spanAttributes, spanAttribute)

		return true
	})

	resourceAttributes.Range(func(key string, value pcommon.Value) bool {
		tagMap[key] = value.AsString()
		spanAttribute := SpanAttribute{
			Key:         key,
			TagType:     "resource",
			IsColumn:    false,
			DataType:    "",
			StringValue: "",
			NumberValue: 0,
		}
		resourceAttrs[key] = value.AsString()
		//nolint:exhaustive
		switch value.Type() {
		case pcommon.ValueTypeDouble:
			numberTagMap[key] = value.Double()
			spanAttribute.NumberValue = value.Double()
			spanAttribute.DataType = "float64"
		case pcommon.ValueTypeInt:
			numberTagMap[key] = float64(value.Int())
			spanAttribute.NumberValue = float64(value.Int())
			spanAttribute.DataType = "float64"
		case pcommon.ValueTypeBool:
			boolTagMap[key] = value.Bool()
			spanAttribute.DataType = "bool"
		default:
			stringTagMap[key] = value.AsString()
			spanAttribute.StringValue = value.AsString()
			spanAttribute.DataType = "string"
		}
		spanAttributes = append(spanAttributes, spanAttribute)

		return true
	})

	references := makeJaegerProtoReferences(otelSpan.Links(), otelSpan.ParentSpanID(), otelSpan.TraceID())

	tenant := "default"
	if tenantAttr, ok := resource.Attributes().Get("tenant"); ok {
		tenant = tenantAttr.AsString()
	}

	//nolint:exhaustruct
	span := &Span{
		TraceID:           TraceIDToHexOrEmptyString(otelSpan.TraceID()),
		SpanID:            SpanIDToHexOrEmptyString(otelSpan.SpanID()),
		ParentSpanID:      SpanIDToHexOrEmptyString(otelSpan.ParentSpanID()),
		Name:              otelSpan.Name(),
		StartTimeUnixNano: uint64(otelSpan.StartTimestamp()),
		DurationNano:      durationNano,
		ServiceName:       serviceName,
		Kind:              int8(otelSpan.Kind()),
		StatusCode:        int16(otelSpan.Status().Code()),
		TagMap:            tagMap,
		StringTagMap:      stringTagMap,
		NumberTagMap:      numberTagMap,
		BoolTagMap:        boolTagMap,
		ResourceTagsMap:   resourceAttrs,
		HasError:          otelSpan.Status().Code() == ptrace.StatusCodeError,
		TraceModel: TraceModel{
			TraceID:           TraceIDToHexOrEmptyString(otelSpan.TraceID()),
			SpanID:            SpanIDToHexOrEmptyString(otelSpan.SpanID()),
			Name:              otelSpan.Name(),
			DurationNano:      durationNano,
			StartTimeUnixNano: uint64(otelSpan.StartTimestamp()),
			ServiceName:       serviceName,
			Kind:              int8(otelSpan.Kind()),
			References:        references,
			TagMap:            tagMap,
			StringTagMap:      stringTagMap,
			NumberTagMap:      numberTagMap,
			BoolTagMap:        boolTagMap,
			HasError:          false,
		},
		Tenant: &tenant,
	}

	attributes.Range(func(key string, value pcommon.Value) bool {
		switch {
		case key == string(semconv.HTTPResponseStatusCodeKey):
			// Handle both string/int http status codes.
			statusString, err := strconv.Atoi(value.Str())
			statusInt := value.Int()
			if err == nil && statusString != 0 {
				statusInt = int64(statusString)
			}
			//nolint:gomnd
			if statusInt >= 400 {
				span.HasError = true
			}
			span.HTTPCode = strconv.FormatInt(statusInt, 10)
			span.ResponseStatusCode = span.HTTPCode
		case key == string(semconv.URLFullKey) && span.Kind == int8(ptrace.SpanKindClient):
			val := value.Str()
			valueURL, err := url.Parse(val)
			if err == nil {
				val = valueURL.Hostname()
			}
			span.ExternalHTTPURL = val
			span.HTTPURL = value.Str()
		case key == string(semconv.HTTPRequestMethodKey) && span.Kind == int8(ptrace.SpanKindClient):
			span.ExternalHTTPMethod = value.Str()
			span.HTTPMethod = value.Str()
		case key == string(semconv.URLFullKey) && span.Kind != int8(ptrace.SpanKindClient):
			span.HTTPURL = value.Str()
		case key == string(semconv.HTTPRequestMethodKey) && span.Kind != int8(ptrace.SpanKindClient):
			span.HTTPMethod = value.Str()
		case key == string(semconv.HTTPRouteKey):
			span.HTTPRoute = value.Str()
		case key == string(semconv.ServerAddressKey):
			span.HTTPHost = value.Str()
		case key == string(semconv.MessagingSystemKey):
			span.MsgSystem = value.Str()
		case key == string(semconv.MessagingOperationKey):
			span.MsgOperation = value.Str()
		case key == string(semconv.DBSystemKey):
			span.DBSystem = value.Str()
		case key == string(semconv.DBNameKey):
			span.DBName = value.Str()
		case key == string(semconv.DBOperationKey):
			span.DBOperation = value.Str()
		case key == string(semconv.PeerServiceKey):
			span.PeerService = value.Str()
		case key == string(semconv.RPCGRPCStatusCodeKey):
			statusString, err := strconv.Atoi(value.Str())
			statusInt := value.Int()
			if err == nil && statusString != 0 {
				statusInt = int64(statusString)
			}
			//nolint:gomnd
			if statusInt > 2 {
				span.HasError = true
			}
			span.GRPCCode = strconv.FormatInt(statusInt, 10)
			span.ResponseStatusCode = span.GRPCCode
		case key == string(semconv.RPCMethodKey):
			span.RPCMethod = value.Str()
			system, found := attributes.Get(string(semconv.RPCSystemKey))
			if found && system.Str() == "grpc" {
				span.GRPCMethod = value.Str()
			}
		case key == string(semconv.RPCServiceKey):
			span.RPCService = value.Str()
		case key == string(semconv.RPCSystemKey):
			span.RPCSystem = value.Str()
		case key == string(semconv.RPCJsonrpcErrorCodeKey):
			span.ResponseStatusCode = value.Str()
		}

		return true
	})

	events := otelSpan.Events()
	for i := 0; i < events.Len(); i++ {
		otelEvent := events.At(i)
		event := Event{
			Name:         otelEvent.Name(),
			TimeUnixNano: uint64(otelEvent.Timestamp()),
			AttributeMap: make(map[string]string),
			IsError:      false,
		}

		otelEvent.Attributes().Range(func(k string, v pcommon.Value) bool {
			event.AttributeMap[k] = v.AsString()

			return true
		})

		if event.Name == "exception" {
			event.IsError = true
			span.ErrorEvent = event
			uuidWithHyphen := uuid.New()
			uuid := strings.ReplaceAll(uuidWithHyphen.String(), "-", "")
			span.ErrorID = uuid
			// TODO: decide whether to do low cardinality exception grouping, default to false
			//nolint:gosec
			hash := md5.Sum([]byte(
				span.ServiceName +
					span.ErrorEvent.AttributeMap[string(semconv.ExceptionTypeKey)] +
					span.ErrorEvent.AttributeMap[string(semconv.ExceptionMessageKey)]))
			span.ErrorGroupID = fmt.Sprintf("%x", hash)
		}

		stringEvent, err := json.Marshal(event)
		if err == nil {
			span.Events = append(span.Events, string(stringEvent))
		}
	}

	span.TraceModel.Events = span.Events
	span.TraceModel.HasError = span.HasError

	spanAttributes = append(spanAttributes, extractSpanAttributesFromSpanIndex(span)...)
	span.SpanAttributes = spanAttributes

	return span
}

func (s *signozExporter) convertTraces(ctx context.Context, traces ptrace.Traces) error {
	resourceSpans := traces.ResourceSpans()

	var batchOfSpans []*Span

	for i := 0; i < resourceSpans.Len(); i++ {
		resourceSpan := resourceSpans.At(i)
		serviceNameStr := "unknown_service"

		serviceName, found := resourceSpan.Resource().Attributes().Get(string(semconv.ServiceNameKey))
		if found && serviceName.Str() != "" {
			serviceNameStr = serviceName.Str()
		}

		scopeSpans := resourceSpan.ScopeSpans()
		for j := 0; j < scopeSpans.Len(); j++ {
			scopeSpan := scopeSpans.At(j)

			spans := scopeSpan.Spans()
			for k := 0; k < spans.Len(); k++ {
				span := spans.At(k)
				batchOfSpans = append(batchOfSpans, s.convertSpan(span, serviceNameStr, resourceSpan.Resource()))
			}
		}
	}

	err := s.writeBatchOfSpans(ctx, batchOfSpans)
	if err != nil {
		return fmt.Errorf("error batch writing spans: %w", err)
	}

	return nil
}

func (s *signozExporter) writeIndexBatch(ctx context.Context, batchSpans []*Span) error {
	statement, err := s.conn.PrepareBatch(ctx, fmt.Sprintf("INSERT INTO %s.%s", traceDB, indexTable))
	if err != nil {
		return fmt.Errorf("could not prepare spans: %w", err)
	}

	for _, span := range batchSpans {
		err = statement.Append(
			time.Unix(0, int64(span.StartTimeUnixNano)),
			span.TraceID,
			span.SpanID,
			span.ParentSpanID,
			span.ServiceName,
			span.Name,
			span.Kind,
			span.DurationNano,
			span.StatusCode,
			span.ExternalHTTPMethod,
			span.ExternalHTTPURL,
			span.Component,
			span.DBSystem,
			span.DBName,
			span.DBOperation,
			span.PeerService,
			span.Events,
			span.HTTPMethod,
			span.HTTPURL,
			span.HTTPCode,
			span.HTTPRoute,
			span.HTTPHost,
			span.MsgSystem,
			span.MsgOperation,
			span.HasError,
			span.TagMap,
			span.GRPCMethod,
			span.GRPCCode,
			span.RPCSystem,
			span.RPCService,
			span.RPCMethod,
			span.ResponseStatusCode,
			span.StringTagMap,
			span.NumberTagMap,
			span.BoolTagMap,
			span.ResourceTagsMap,
		)
		if err != nil {
			return fmt.Errorf("could not append span: %w", err)
		}
	}

	err = statement.Send()
	if err != nil {
		return fmt.Errorf("could not send spans: %w", err)
	}

	return nil
}

var errInvalidSpan = fmt.Errorf("invalid span")

func (s *signozExporter) writeTagBatch(ctx context.Context, batchSpans []*Span) error { //nolint:funlen,cyclop
	statement, err := s.conn.PrepareBatch(ctx, fmt.Sprintf("INSERT INTO %s.%s", traceDB, attributeTable))
	if err != nil {
		return fmt.Errorf("could not prepare tags: %w", err)
	}

	statementKeys, err := s.conn.PrepareBatch(ctx, fmt.Sprintf("INSERT INTO %s.%s", traceDB, attributeKeyTable))
	if err != nil {
		return fmt.Errorf("could not prepare tag keys: %w", err)
	}

	spanAttributeKeySet := make(map[SpanAttributeKey]struct{})
	spanAttributeSet := make(map[SpanAttribute]struct{})

	for _, span := range batchSpans {
		for _, spanAttribute := range span.SpanAttributes {
			if _, ok := spanAttributeSet[spanAttribute]; ok {
				continue
			} else {
				spanAttributeSet[spanAttribute] = struct{}{}
			}

			spanAttributeKey := spanAttribute.ToKey()
			if _, ok := spanAttributeKeySet[spanAttributeKey]; !ok {
				err = statementKeys.Append(
					spanAttributeKey.Key,
					spanAttributeKey.TagType,
					spanAttributeKey.DataType,
					spanAttributeKey.IsColumn,
				)
				if err != nil {
					return fmt.Errorf("could not append tag key: %w", err)
				}
			} else {
				spanAttributeKeySet[spanAttributeKey] = struct{}{}
			}

			switch spanAttribute.DataType {
			case "string":
				err = statement.Append(
					time.Unix(0, int64(span.StartTimeUnixNano)),
					spanAttribute.Key,
					spanAttribute.TagType,
					spanAttribute.DataType,
					spanAttribute.StringValue,
					nil,
					spanAttribute.IsColumn,
				)
			case "float64":
				err = statement.Append(
					time.Unix(0, int64(span.StartTimeUnixNano)),
					spanAttribute.Key,
					spanAttribute.TagType,
					spanAttribute.DataType,
					nil,
					spanAttribute.NumberValue,
					spanAttribute.IsColumn,
				)
			case "bool":
				err = statement.Append(
					time.Unix(0, int64(span.StartTimeUnixNano)),
					spanAttribute.Key,
					spanAttribute.TagType,
					spanAttribute.DataType,
					nil,
					nil,
					spanAttribute.IsColumn,
				)
			default:
				err = fmt.Errorf("unknown span attribute type %s: %w", spanAttribute.DataType, errInvalidSpan)
			}

			if err != nil {
				return fmt.Errorf("could not append tag: %w", err)
			}
		}
	}

	err = statement.Send()
	if err != nil {
		return fmt.Errorf("could not send tags: %w", err)
	}

	err = statementKeys.Send()
	if err != nil {
		return fmt.Errorf("could not send tag keys: %w", err)
	}

	return nil
}

func (s *signozExporter) writeErrorBatch(ctx context.Context, batchSpans []*Span) error {
	statement, err := s.conn.PrepareBatch(ctx, fmt.Sprintf("INSERT INTO %s.%s", traceDB, errorTable))
	if err != nil {
		return fmt.Errorf("could not prepare errors: %w", err)
	}

	for _, span := range batchSpans {
		if span.ErrorEvent.Name == "" {
			continue
		}

		err = statement.Append(
			time.Unix(0, int64(span.ErrorEvent.TimeUnixNano)),
			span.ErrorID,
			span.ErrorGroupID,
			span.TraceID,
			span.SpanID,
			span.ServiceName,
			span.ErrorEvent.AttributeMap["exception.type"],
			span.ErrorEvent.AttributeMap["exception.message"],
			span.ErrorEvent.AttributeMap["exception.stacktrace"],
			strings.ToLower(span.ErrorEvent.AttributeMap["exception.escaped"]) == "true",
			span.ResourceTagsMap,
		)
		if err != nil {
			return fmt.Errorf("could not append error: %w", err)
		}
	}

	err = statement.Send()
	if err != nil {
		return fmt.Errorf("could not send errors: %w", err)
	}

	return nil
}

type Metric struct {
	Size  int64
	Count int64
}

func (s *signozExporter) writeModelBatch(ctx context.Context, batchSpans []*Span) error {
	statement, err := s.conn.PrepareBatch(ctx, fmt.Sprintf("INSERT INTO %s.%s", traceDB, spansTable))
	if err != nil {
		return fmt.Errorf("could not prepare trace models: %w", err)
	}

	for _, span := range batchSpans {
		usageMap := span.TraceModel
		usageMap.TagMap = map[string]string{}

		serialized, err := json.Marshal(span.TraceModel)
		if err != nil {
			return fmt.Errorf("could not marshal trace model: %w", err)
		}

		err = statement.Append(time.Unix(0, int64(span.StartTimeUnixNano)), span.TraceID, string(serialized))
		if err != nil {
			return fmt.Errorf("could not append trace model: %w", err)
		}
	}

	err = statement.Send()
	if err != nil {
		return fmt.Errorf("could not send trace models: %w", err)
	}

	return nil
}

func (s *signozExporter) writeBatchOfSpans(ctx context.Context, batchSpans []*Span) error {
	err := s.writeIndexBatch(ctx, batchSpans)
	if err != nil {
		return fmt.Errorf("could not write index batch: %w", err)
	}

	err = s.writeTagBatch(ctx, batchSpans)
	if err != nil {
		return fmt.Errorf("could not write tag batch: %w", err)
	}

	err = s.writeErrorBatch(ctx, batchSpans)
	if err != nil {
		return fmt.Errorf("could not write error batch: %w", err)
	}

	err = s.writeModelBatch(ctx, batchSpans)
	if err != nil {
		return fmt.Errorf("could not write model batch: %w", err)
	}

	return nil
}
