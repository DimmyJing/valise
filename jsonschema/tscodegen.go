package jsonschema

import (
	"fmt"
	"slices"
	"strings"
)

func JSONSchemaToTS(inp *JSONInfo, prefix string) (string, error) {
	types, err := jsonSchemaToTS(*inp)
	if err != nil {
		return "", fmt.Errorf("failed to convert json schema to ts: %w", err)
	}

	return FormatComment(inp.Description) + prefix + types, nil
}

var errInvalidSchema = fmt.Errorf("invalid schema")

func FormatComment(comment string) string {
	if comment == "" {
		return ""
	}

	commentLines := strings.Split(comment, "\n")

	if len(commentLines) == 1 {
		return fmt.Sprintf("/** %s */\n", commentLines[0])
	}

	var builder strings.Builder

	builder.WriteString("/**\n")

	for _, line := range commentLines {
		builder.WriteString(fmt.Sprintf(" * %s\n", line))
	}

	builder.WriteString(" */\n")

	return builder.String()
}

func indentMiddle(input string) string {
	inputSplit := strings.Split(input, "\n")
	//nolint:gomnd
	if len(inputSplit) < 3 {
		return input
	}

	var builder strings.Builder

	for idx, line := range inputSplit {
		switch idx {
		case 0:
			builder.WriteString(line + "\n")
		case len(inputSplit) - 1:
			builder.WriteString(line)
		default:
			builder.WriteString("  " + line + "\n")
		}
	}

	return builder.String()
}

func jsonSchemaToTS(input JSONInfo) (string, error) { //nolint:funlen,cyclop,gocognit
	if input.Type == "" {
		return "unknown", nil
	}

	switch input.Type {
	case "string":
		if len(input.Enums) > 0 {
			return "\"" + strings.Join(input.Enums, "\" | \"") + "\"", nil
		}

		switch input.Format {
		case "duration":
			return "Duration", nil
		case "date-time":
			return "Date", nil
		default:
			return "string", nil
		}
	case "number", "integer":
		return "number", nil
	case "boolean":
		return "boolean", nil
	case "array":
		res, err := jsonSchemaToTS(*input.Items)
		if err != nil {
			return "", fmt.Errorf("failed to convert array items: %w", err)
		}

		return res + "[]", nil
	case "null":
		return "null", nil
	case "object":
		//nolint:nestif
		if prop, ok := input.AdditionalProperties.(*bool); ok {
			if *prop {
				return "Record<string, any>", nil
			}

			if input.Properties.Len() == 0 {
				return "Record<string, never>", nil
			}

			var insideBuilder strings.Builder

			insideBuilder.WriteString("{\n")

			for pair := input.Properties.Oldest(); pair != nil; pair = pair.Next() {
				key, value := pair.Key, pair.Value
				required := slices.Index(input.Required, key) != -1

				var optional string

				// TODO: implement oneOf
				if !required {
					optional = "?"
				}

				res, err := jsonSchemaToTS(*value)
				if err != nil {
					return "", fmt.Errorf("failed to convert object properties: %w", err)
				}

				insideBuilder.WriteString(FormatComment(value.Description))
				insideBuilder.WriteString(fmt.Sprintf("%s%s: %s;\n", key, optional, res))
			}
			insideBuilder.WriteString("}")

			return indentMiddle(insideBuilder.String()), nil
		} else if prop, ok := input.AdditionalProperties.(JSONInfo); ok {
			res, err := jsonSchemaToTS(prop)
			if err != nil {
				return "", fmt.Errorf("failed to convert object properties: %w", err)
			}

			return fmt.Sprintf("Record<string, " + res + ">"), nil
		}
	}

	return "", fmt.Errorf("invalid type %s: %w", input.Type, errInvalidSchema)
}
