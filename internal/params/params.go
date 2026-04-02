package params

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"reflect"
	"time"
)

// Normalize converts arbitrary Go values into database/sql driver values.
func Normalize(value any) (driver.Value, error) {
	if value == nil {
		return nil, nil
	}

	if valuer, ok := value.(driver.Valuer); ok {
		out, err := valuer.Value()
		if err != nil {
			return nil, err
		}
		return Normalize(out)
	}

	switch typed := value.(type) {
	case time.Time:
		return typed, nil
	case *time.Time:
		if typed == nil {
			return nil, nil
		}
		return *typed, nil
	case []byte:
		return typed, nil
	case json.RawMessage:
		return string(typed), nil
	}

	if converted, err := driver.DefaultParameterConverter.ConvertValue(value); err == nil {
		return converted, nil
	}

	rv := reflect.ValueOf(value)
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil, nil
		}
		rv = rv.Elem()
	}

	switch rv.Kind() {
	case reflect.Map, reflect.Struct, reflect.Slice, reflect.Array:
		if rv.Kind() == reflect.Slice && rv.Type().Elem().Kind() == reflect.Uint8 {
			return rv.Bytes(), nil
		}
		payload, err := json.Marshal(rv.Interface())
		if err != nil {
			return nil, fmt.Errorf("marshal parameter: %w", err)
		}
		return string(payload), nil
	default:
		return nil, fmt.Errorf("unsupported parameter type %T", value)
	}
}

// NormalizeAll converts a value slice into driver values.
func NormalizeAll(values []any) ([]driver.Value, error) {
	out := make([]driver.Value, len(values))
	for index := range values {
		converted, err := Normalize(values[index])
		if err != nil {
			return nil, err
		}
		out[index] = converted
	}
	return out, nil
}
