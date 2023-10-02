package jsonschema

import (
	"errors"
	"fmt"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ValueToAny(value protoreflect.Value, fieldDesc protoreflect.FieldDescriptor) any {
	switch val := value.Interface().(type) {
	case protoreflect.Message:
		return MessageToAny(val)
	case protoreflect.List:
		res := []any{}
		for i := 0; i < val.Len(); i++ {
			res = append(res, ValueToAny(val.Get(i), fieldDesc))
		}

		return res
	case protoreflect.Map:
		res := make(map[string]any)

		val.Range(func(mk protoreflect.MapKey, v protoreflect.Value) bool {
			res[mk.String()] = ValueToAny(v, fieldDesc)

			return true
		})

		return res
	case protoreflect.EnumNumber:
		if fieldDesc.Enum().FullName() == "NullValue" {
			return nil
		}

		return string(fieldDesc.Enum().Values().ByNumber(val).Name())
	case nil, bool, int32, int64, uint32, uint64, float32, float64, string, []byte:
		return val
	}

	return nil
}

func MessageToAny(message protoreflect.Message) any {
	res := make(map[string]any)

	messageDesc := message.Descriptor()
	messageFullName := messageDesc.FullName()
	//nolint:goconst
	if messageFullName.Parent() == "google.protobuf" {
		switch msg := message.Interface().(type) {
		case *timestamppb.Timestamp:
			if msg == nil {
				return time.Time{}
			}

			return msg.AsTime()
		case *durationpb.Duration:
			return msg.AsDuration()
		case *emptypb.Empty:
			return make(map[string]struct{})
		case *structpb.Struct:
			return msg.AsMap()
		case *structpb.Value:
			return msg.AsInterface()
		}
	}

	if message == nil {
		return res
	}

	fields := message.Descriptor().Fields()
	for i := 0; i < fields.Len(); i++ {
		fd := fields.Get(i)
		res[fd.JSONName()] = ValueToAny(message.Get(fd), fd)
	}

	return res
}

var errInvalidMessage = errors.New("invalid message")

func AnyToScalar(input any, fieldDesc protoreflect.FieldDescriptor) (protoreflect.Value, error) {
	// TODO: add support for more types
	//nolint:exhaustive
	switch fieldDesc.Kind() {
	case protoreflect.EnumKind:
		enumDesc := fieldDesc.Enum()

		inputStr, ok := input.(string)
		if !ok {
			return protoreflect.ValueOf(nil), fmt.Errorf("invalid enum value %v: %w", input, errInvalidMessage)
		}

		enumVal := enumDesc.Values().ByName(protoreflect.Name(inputStr))
		if enumVal == nil {
			return protoreflect.ValueOf(nil), fmt.Errorf("invalid enum value %v: %w", input, errInvalidMessage)
		} else {
			return protoreflect.ValueOfEnum(enumVal.Number()), nil
		}
	default:
		return protoreflect.ValueOf(input), nil
	}
}

func AnyToRepeated( //nolint:funlen,gocognit,cyclop
	input any,
	value protoreflect.Value,
	fieldDesc protoreflect.FieldDescriptor,
) error {
	switch val := value.Interface().(type) {
	case protoreflect.Message:
		return AnyToMessage(input, val)
	case protoreflect.List:
		inp, ok := input.([]any)
		if !ok {
			return fmt.Errorf("input is not list %v: %w", input, errInvalidMessage)
		}

		if fieldDesc.Kind() == protoreflect.MessageKind {
			for _, v := range inp {
				valEl := val.NewElement()

				err := AnyToMessage(v, valEl.Message())
				if err != nil {
					return fmt.Errorf("failed to convert list element %v: %w", v, err)
				}

				val.Append(valEl)
			}
		} else {
			for _, v := range inp {
				valEl, err := AnyToScalar(v, fieldDesc)
				if err != nil {
					return fmt.Errorf("failed to convert list element scalar %v: %w", v, err)
				}

				val.Append(valEl)
			}
		}

		return nil
	case protoreflect.Map:
		inp, ok := input.(map[string]any)
		if !ok {
			return fmt.Errorf("input is not map %v: %w", input, errInvalidMessage)
		}

		isMsg := fieldDesc.Kind() == protoreflect.MessageKind

		for key, valEl := range inp {
			if isMsg {
				valEl := val.NewValue()

				err := AnyToMessage(valEl, valEl.Message())
				if err != nil {
					return fmt.Errorf("failed to convert map element %v: %w", valEl, err)
				}

				val.Set(protoreflect.ValueOfString(key).MapKey(), valEl)
			} else {
				valEl, err := AnyToScalar(valEl, fieldDesc)
				if err != nil {
					return fmt.Errorf("failed to convert map element scalar %v: %w", valEl, err)
				}

				val.Set(protoreflect.ValueOfString(key).MapKey(), valEl)
			}
		}
	}

	return fmt.Errorf("unsupported repeated type %v: %w", value.Interface(), errInvalidMessage)
}

func AnyToMessage(input any, msg protoreflect.Message) error { //nolint:cyclop,funlen,gocognit
	messageDesc := msg.Descriptor()

	messageFullName := messageDesc.FullName()
	//nolint:nestif
	if messageFullName.Parent() == "google.protobuf" {
		switch messageFullName.Name() {
		case "Timestamp":
			if inp, ok := input.(time.Time); ok {
				if inp.IsZero() {
					newTime := &timestamppb.Timestamp{
						Seconds: -62135596800,
						Nanos:   0,
					}
					proto.Merge(msg.Interface(), newTime)
				} else {
					proto.Merge(msg.Interface(), timestamppb.New(inp))
				}

				return nil
			} else {
				return fmt.Errorf("input is not time.Time %v: %w", input, errInvalidMessage)
			}
		//nolint:goconst
		case "Duration":
			if inp, ok := input.(time.Duration); ok {
				proto.Merge(msg.Interface(), durationpb.New(inp))

				return nil
			} else {
				return fmt.Errorf("input is not time.Duration %v: %w", input, errInvalidMessage)
			}
		case "Empty":
			proto.Merge(msg.Interface(), &emptypb.Empty{})

			return nil
		case "Struct":
			inp, ok := input.(map[string]any)
			if !ok {
				return fmt.Errorf("input is not map %v: %w", input, errInvalidMessage)
			}

			res, err := structpb.NewStruct(inp)
			if err != nil {
				return fmt.Errorf("failed to convert struct %v: %w", inp, err)
			}

			proto.Merge(msg.Interface(), res)

			return nil
		case "Value":
			res, err := structpb.NewValue(input)
			if err != nil {
				return fmt.Errorf("failed to convert value %v: %w", input, err)
			}

			proto.Merge(msg.Interface(), res)

			return nil
		}
	}

	inp, ok := input.(map[string]any)
	if !ok {
		return fmt.Errorf("input is not map %v: %w", input, errInvalidMessage)
	}

	var err error

	oneOfs := map[int]bool{}

	fields := messageDesc.Fields()
	for i := 0; i < fields.Len(); i++ {
		fieldDesc := fields.Get(i)

		rawInp, ok := inp[fieldDesc.JSONName()]
		if !ok {
			continue
		}

		switch {
		case fieldDesc.IsList(), fieldDesc.IsMap():
			val := msg.NewField(fieldDesc)
			err = AnyToRepeated(rawInp, val, fieldDesc)
			msg.Set(fieldDesc, val)
		default:
			if od := fieldDesc.ContainingOneof(); od != nil {
				if oneOfs[od.Index()] {
					return fmt.Errorf("duplicate oneof field %s: %w", fieldDesc.JSONName(), errInvalidMessage)
				}

				oneOfs[od.Index()] = true
			}

			if fieldDesc.Kind() == protoreflect.MessageKind {
				val := msg.NewField(fieldDesc)
				err = AnyToMessage(rawInp, val.Message())
				msg.Set(fieldDesc, val)
			} else {
				var val protoreflect.Value
				val, err = AnyToScalar(rawInp, fieldDesc)
				msg.Set(fieldDesc, val)
			}
		}

		if err != nil {
			return fmt.Errorf("failed to convert field %s-%v to message: %w", fieldDesc.JSONName(), rawInp, err)
		}
	}

	if err != nil {
		return err
	}

	return nil
}
