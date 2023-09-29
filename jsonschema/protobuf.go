package jsonschema

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/DimmyJing/valise/log"
	orderedmap "github.com/wk8/go-ordered-map/v2"
	"github.com/yoheimuta/go-protoparser"
	"github.com/yoheimuta/go-protoparser/parser"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type jsonInfo struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`

	Type   string `json:"type,omitempty"`
	Format string `json:"format,omitempty"`

	Enums []string `json:"enum,omitempty"`

	Items *jsonInfo `json:"items,omitempty"`

	Properties *orderedmap.OrderedMap[string, *jsonInfo] `json:"properties,omitempty"`
	Required   []string                                  `json:"reqired,omitempty"`

	OneOf []*jsonInfo `json:"oneOf,omitempty"`
	AllOf []*jsonInfo `json:"allOf,omitempty"`

	AdditionalProperties any `json:"additionalProperties,omitempty"`

	optional bool
}

var errInvalidField = fmt.Errorf("invalid field")

func generateFromField(field protoreflect.FieldDescriptor, nested bool) *jsonInfo { //nolint:funlen,cyclop
	//nolint:exhaustruct
	info := jsonInfo{
		Title:       field.JSONName(),
		Description: protoMap[string(field.FullName())],
		optional:    field.HasOptionalKeyword(),
	}

	if nested {
		info.Title = ""
		info.Description = ""
	}

	if field.IsList() && !nested {
		info.Type = "array"
		info.Items = generateFromField(field, true)

		return &info
	}

	if field.IsMap() {
		if field.MapKey().Kind() != protoreflect.StringKind {
			log.Panic(fmt.Errorf("invalid map key type %T: %w", field.MapKey().Kind(), errInvalidField))
		}

		//nolint:goconst
		info.Type = "object"
		info.AdditionalProperties = generateFromField(field.MapValue(), false)

		return &info
	}

	switch field.Kind() {
	case protoreflect.BoolKind:
		//nolint:goconst
		info.Type = "boolean"
	case protoreflect.EnumKind:
		//nolint:goconst
		info.Type = "string"

		enumDescriptor := field.Enum()

		enumValues := enumDescriptor.Values()
		for i := 0; i < enumValues.Len(); i++ {
			info.Enums = append(info.Enums, string(enumValues.Get(i).Name()))
		}
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Uint32Kind,
		protoreflect.Sfixed32Kind, protoreflect.Fixed32Kind:
		//nolint:goconst
		info.Type = "integer"
		info.Format = "int32"
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Uint64Kind,
		protoreflect.Sfixed64Kind, protoreflect.Fixed64Kind:
		info.Type = "integer"
		info.Format = "int64"
	case protoreflect.FloatKind:
		//nolint:goconst
		info.Type = "number"
		info.Format = "float"
	case protoreflect.DoubleKind:
		info.Type = "number"
		info.Format = "double"
	case protoreflect.StringKind:
		info.Type = "string"
	case protoreflect.BytesKind:
		info.Type = "string"
	case protoreflect.MessageKind:
		info.Type = "object"

		title, description, optional := info.Title, info.Description, info.optional
		info = *generateFromMessage(field.Message())
		info.Title = title
		info.optional = optional

		if info.Description == "" {
			info.Description = description
		}
	case protoreflect.GroupKind:
		log.Panic(fmt.Errorf("group kind is not supported: %w", errInvalidField))
	}

	return &info
}

//nolint:gochecknoglobals
var wellKnownTypes = map[string]bool{
	"Timestamp": true,
	"Duration":  true,
	"Empty":     true,
	"Struct":    true,
	"Value":     true,
}

func generateFromMessage(message protoreflect.MessageDescriptor) *jsonInfo { //nolint:funlen,cyclop
	//nolint:exhaustruct
	info := jsonInfo{
		Title:       string(message.Name()),
		Description: protoMap[string(message.FullName())],
		Type:        "object",
	}
	falseVal := false
	info.AdditionalProperties = &falseVal

	allOneOfs := [][]*jsonInfo{}
	oneOfNames := map[string]bool{}

	if wellKnownTypes[string(message.Name())] && message.ParentFile().Package() == "google.protobuf" {
		info.AdditionalProperties = nil

		switch message.Name() {
		case "Duration":
			info.Type = "string"
			info.Format = "duration"
		case "Empty":
			info.AdditionalProperties = &falseVal
		case "Struct":
			trueVal := true
			info.AdditionalProperties = &trueVal
		case "Timestamp":
			info.Type = "string"
			info.Format = "date-time"
		case "Value":
			info.Type = ""
		}

		return &info
	}

	msgOneOfs := message.Oneofs()
	for i := 0; i < msgOneOfs.Len(); i++ {
		oneOf := msgOneOfs.Get(i)
		if oneOf.IsSynthetic() {
			continue
		}

		oneOfs := []*jsonInfo{}

		fields := oneOf.Fields()
		for j := 0; j < fields.Len(); j++ {
			name := fields.Get(j).JSONName()
			//nolint:exhaustruct
			oneOfs = append(oneOfs, &jsonInfo{Required: []string{name}})
			oneOfNames[name] = true
		}

		allOneOfs = append(allOneOfs, oneOfs)
	}

	fields := message.Fields()
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)

		fieldInfo := generateFromField(field, false)
		if !fieldInfo.optional && !oneOfNames[fieldInfo.Title] {
			info.Required = append(info.Required, fieldInfo.Title)
		}

		if info.Properties == nil {
			info.Properties = orderedmap.New[string, *jsonInfo]()
		}

		info.Properties.Set(fieldInfo.Title, fieldInfo)
	}

	if len(allOneOfs) > 0 {
		if len(allOneOfs) > 1 {
			info.AllOf = []*jsonInfo{}
			for _, oneOfs := range allOneOfs {
				//nolint:exhaustruct
				info.AllOf = append(info.AllOf, &jsonInfo{OneOf: oneOfs})
			}
		} else {
			info.OneOf = allOneOfs[0]
		}
	}

	return &info
}

//nolint:gochecknoglobals
var protoMap = make(map[string]string)

var errNoPackageName = fmt.Errorf("no package names in proto file")

func processComments(
	name string,
	prefix string,
	comments []*parser.Comment,
	inlineComment *parser.Comment,
	inlineComment2 *parser.Comment,
) {
	allComments := []*parser.Comment{}
	allComments = append(allComments, comments...)
	allComments = append(allComments, inlineComment)
	allComments = append(allComments, inlineComment2)
	lines := []string{}

	for _, comment := range allComments {
		if comment == nil {
			continue
		}

		addLines := comment.Lines()
		if len(addLines) > 0 && len(lines) > 0 {
			lines = append(lines, "")
		}

		for _, line := range addLines {
			lines = append(lines, strings.TrimSpace(line))
		}
	}

	protoMap[prefix+"."+name] = strings.Join(lines, "\n")
}

func processVisitee(visitee parser.Visitee, prefix string) { //nolint:cyclop,funlen
	switch visitee := visitee.(type) {
	case *parser.Enum:
		processComments(
			visitee.EnumName,
			prefix,
			visitee.Comments,
			visitee.InlineComment,
			visitee.InlineCommentBehindLeftCurly,
		)
	case *parser.EnumField:
		processComments(visitee.Ident, prefix, visitee.Comments, visitee.InlineComment, nil)
	case *parser.Extend:
		processComments(
			visitee.MessageType,
			prefix,
			visitee.Comments,
			visitee.InlineComment,
			visitee.InlineCommentBehindLeftCurly,
		)

		for _, v := range visitee.ExtendBody {
			processVisitee(v, prefix+"."+visitee.MessageType)
		}
	case *parser.Field:
		processComments(visitee.FieldName, prefix, visitee.Comments, visitee.InlineComment, nil)
	case *parser.MapField:
		processComments(visitee.MapName, prefix, visitee.Comments, visitee.InlineComment, nil)
	case *parser.Message:
		processComments(
			visitee.MessageName,
			prefix,
			visitee.Comments,
			visitee.InlineComment,
			visitee.InlineCommentBehindLeftCurly,
		)

		for _, v := range visitee.MessageBody {
			processVisitee(v, prefix+"."+visitee.MessageName)
		}
	case *parser.Oneof:
		processComments(
			visitee.OneofName,
			prefix,
			visitee.Comments,
			visitee.InlineComment,
			visitee.InlineCommentBehindLeftCurly,
		)

		for _, o := range visitee.OneofFields {
			processVisitee(o, prefix)
		}

		for _, o := range visitee.Options {
			processVisitee(o, prefix+"."+visitee.OneofName)
		}
	case *parser.OneofField:
		processComments(visitee.FieldName, prefix, visitee.Comments, visitee.InlineComment, nil)
	case *parser.Option:
		processComments(visitee.OptionName, prefix, visitee.Comments, visitee.InlineComment, nil)
	case *parser.Package:
		processComments(visitee.Name, prefix, visitee.Comments, visitee.InlineComment, nil)
	}
}

func processParsedProto(parsed *parser.Proto) error {
	packageName := ""

	for _, visitee := range parsed.ProtoBody {
		if pkg, ok := visitee.(*parser.Package); ok {
			packageName = pkg.Name
		}
	}

	if packageName == "" {
		return errNoPackageName
	}

	for _, visitee := range parsed.ProtoBody {
		processVisitee(visitee, packageName)
	}

	return nil
}

func InitProtoMap(projectDir string) {
	paths := []string{}

	err := filepath.WalkDir(projectDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if filepath.Ext(d.Name()) != ".proto" {
			return nil
		}

		paths = append(paths, path)

		return nil
	})
	if err != nil {
		log.Warnf("cannot walk project directory %s: %v", projectDir, err)
	}

	for _, path := range paths {
		file, err := os.Open(path)
		if err != nil {
			log.Warnf("cannot open possible proto path %s: %v", path, err)
		}

		proto, err := protoparser.Parse(file)
		if err != nil {
			log.Warnf("cannot parse possible proto file %s: %v", path, err)
		}

		err = processParsedProto(proto)
		if err != nil {
			log.Warnf("cannot process possible proto file %s: %v", path, err)
		}
	}
}

func GenerateSchema(message proto.Message) string {
	info := generateFromMessage(message.ProtoReflect().Descriptor())

	res, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		log.Panic(err)
	}

	return string(res)
}
