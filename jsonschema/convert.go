package jsonschema

import (
	"errors"
	"fmt"
	"time"

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
	res := make(map[any]any)

	messageDesc := message.Descriptor()
	messageFullName := messageDesc.FullName()
	//nolint:goconst
	if messageFullName.Parent() == "google.protobuf" {
		switch msg := message.Interface().(type) {
		case *timestamppb.Timestamp:
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

	message.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		res[fd.JSONName()] = ValueToAny(v, fd)

		return true
	})

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

		//nolint:nestif
		if fieldDesc.Kind() == protoreflect.MessageKind {
			for _, v := range inp {
				valEl := val.NewElement()

				err := AnyToMessage(v, valEl.Message())
				if err != nil {
					return fmt.Errorf("failed to convert list element %v: %w", v, err)
				}

				if valEl.IsValid() {
					val.Append(valEl)
				}
			}
		} else {
			for _, v := range inp {
				valEl, err := AnyToScalar(v, fieldDesc)
				if err != nil {
					return fmt.Errorf("failed to convert list element scalar %v: %w", v, err)
				}

				if valEl.IsValid() {
					val.Append(valEl)
				}
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
			//nolint:nestif
			if isMsg {
				valEl := val.NewValue()

				err := AnyToMessage(valEl, valEl.Message())
				if err != nil {
					return fmt.Errorf("failed to convert map element %v: %w", valEl, err)
				}

				if valEl.IsValid() {
					val.Set(protoreflect.ValueOfString(key).MapKey(), valEl)
				}
			} else {
				valEl, err := AnyToScalar(valEl, fieldDesc)
				if err != nil {
					return fmt.Errorf("failed to convert map element scalar %v: %w", valEl, err)
				}

				if valEl.IsValid() {
					val.Set(protoreflect.ValueOfString(key).MapKey(), valEl)
				}
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
				msg.SetUnknown(timestamppb.New(inp).ProtoReflect().GetUnknown())

				return nil
			} else {
				return fmt.Errorf("input is not time.Time %v: %w", input, errInvalidMessage)
			}
		//nolint:goconst
		case "Duration":
			if inp, ok := input.(time.Duration); ok {
				msg.SetUnknown(durationpb.New(inp).ProtoReflect().GetUnknown())

				return nil
			} else {
				return fmt.Errorf("input is not time.Duration %v: %w", input, errInvalidMessage)
			}
		case "Empty":
			msg.SetUnknown((&emptypb.Empty{}).ProtoReflect().GetUnknown())

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

			msg.SetUnknown(res.ProtoReflect().GetUnknown())

			return nil
		case "Value":
			res, err := structpb.NewValue(input)
			if err != nil {
				return fmt.Errorf("failed to convert value %v: %w", input, err)
			}

			msg.SetUnknown(res.ProtoReflect().GetUnknown())

			return nil
		}
	}

	inp, ok := input.(map[string]any)
	if !ok {
		return fmt.Errorf("input is not map %v: %w", input, errInvalidMessage)
	}

	var err error

	oneOfs := map[int]bool{}

	msg.Range(func(fieldDesc protoreflect.FieldDescriptor, val protoreflect.Value) bool {
		rawInp, ok := inp[fieldDesc.JSONName()]
		if !ok {
			return true
		}

		switch {
		case fieldDesc.IsList(), fieldDesc.IsMap():
			err = AnyToRepeated(rawInp, val, fieldDesc)
		default:
			if od := fieldDesc.ContainingOneof(); od != nil {
				if oneOfs[od.Index()] {
					err = fmt.Errorf("duplicate oneof field %s: %w", fieldDesc.JSONName(), errInvalidMessage)
				}
				oneOfs[od.Index()] = true
			}

			if fieldDesc.Kind() == protoreflect.MessageKind {
				val := msg.NewField(fieldDesc)
				err = AnyToMessage(rawInp, val.Message())
			} else {
				var val protoreflect.Value
				val, err = AnyToScalar(rawInp, fieldDesc)
				if val.IsValid() {
					msg.Set(fieldDesc, val)
				}
			}
		}
		if err != nil {
			err = fmt.Errorf("failed to convert field %s-%v to message: %w", fieldDesc.JSONName(), rawInp, err)

			return false
		}

		return true
	})

	if err != nil {
		return err
	}

	return nil
}
