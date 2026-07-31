package micromustache

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
)

const (
	maxSafeInteger      = 1<<53 - 1
	maxStringifyNesting = 1000
)

func appendJSValue(result *strings.Builder, value Value, explicit bool) error {
	if value == nil {
		if explicit {
			result.WriteString("null")
		}
		return nil
	}
	if _, ok := value.(Undefined); ok {
		if explicit {
			result.WriteString("undefined")
		}
		return nil
	}
	return appendJSNonNullValue(result, value, nil, 0)
}

func appendJSNonNullValue(result *strings.Builder, value Value, activeSlices map[uintptr]struct{}, depth int) error {
	if depth > maxStringifyNesting {
		return unsupportedValue("array nesting exceeds %d levels", maxStringifyNesting)
	}

	switch value := value.(type) {
	case Undefined:
		return nil
	case nil:
		return nil
	case string:
		result.WriteString(value)
	case bool:
		result.WriteString(strconv.FormatBool(value))
	case float64:
		result.WriteString(formatJSNumber(value, 64))
	case float32:
		result.WriteString(formatJSNumber(float64(value), 32))
	case int:
		return appendSignedInteger(result, int64(value), value)
	case int8:
		result.WriteString(strconv.FormatInt(int64(value), 10))
	case int16:
		result.WriteString(strconv.FormatInt(int64(value), 10))
	case int32:
		result.WriteString(strconv.FormatInt(int64(value), 10))
	case int64:
		return appendSignedInteger(result, value, value)
	case uint:
		return appendUnsignedInteger(result, uint64(value), value)
	case uint8:
		result.WriteString(strconv.FormatUint(uint64(value), 10))
	case uint16:
		result.WriteString(strconv.FormatUint(uint64(value), 10))
	case uint32:
		result.WriteString(strconv.FormatUint(uint64(value), 10))
	case uint64:
		return appendUnsignedInteger(result, value, value)
	case []any:
		return appendJSArray(result, value, activeSlices, depth)
	case Scope:
		result.WriteString("[object Object]")
	case map[string]any:
		result.WriteString("[object Object]")
	default:
		return unsupportedValue("Go type %T", value)
	}
	return nil
}

func appendSignedInteger(result *strings.Builder, value int64, original Value) error {
	if value < -maxSafeInteger || value > maxSafeInteger {
		return unsupportedValue("integer %v is outside JavaScript's safe range", original)
	}
	result.WriteString(strconv.FormatInt(value, 10))
	return nil
}

func appendUnsignedInteger(result *strings.Builder, value uint64, original Value) error {
	if value > maxSafeInteger {
		return unsupportedValue("integer %v is outside JavaScript's safe range", original)
	}
	result.WriteString(strconv.FormatUint(value, 10))
	return nil
}

func appendJSArray(result *strings.Builder, values []any, activeSlices map[uintptr]struct{}, depth int) error {
	if activeSlices == nil {
		activeSlices = make(map[uintptr]struct{})
	}
	identity := reflect.ValueOf(values).Pointer()
	if identity != 0 {
		if _, exists := activeSlices[identity]; exists {
			return unsupportedValue("cyclic array")
		}
		activeSlices[identity] = struct{}{}
		defer delete(activeSlices, identity)
	}

	for index, value := range values {
		if index > 0 {
			result.WriteByte(',')
		}
		if err := appendJSNonNullValue(result, value, activeSlices, depth+1); err != nil {
			return err
		}
	}
	return nil
}

func formatJSNumber(value float64, bitSize int) string {
	switch {
	case math.IsNaN(value):
		return "NaN"
	case math.IsInf(value, 1):
		return "Infinity"
	case math.IsInf(value, -1):
		return "-Infinity"
	case value == 0:
		return "0"
	}

	abs := math.Abs(value)
	if abs >= 1e-6 && abs < 1e21 {
		return strconv.FormatFloat(value, 'f', -1, bitSize)
	}
	formatted := strconv.FormatFloat(value, 'e', -1, bitSize)
	mantissa, exponent, found := strings.Cut(formatted, "e")
	if !found {
		return formatted
	}
	sign := ""
	if len(exponent) > 0 && (exponent[0] == '+' || exponent[0] == '-') {
		sign = exponent[:1]
		exponent = exponent[1:]
	}
	exponent = strings.TrimLeft(exponent, "0")
	if exponent == "" {
		exponent = "0"
	}
	return mantissa + "e" + sign + exponent
}

func unsupportedValue(format string, args ...any) error {
	return fmt.Errorf("%s: %w", fmt.Sprintf(format, args...), ErrUnsupportedValue)
}
