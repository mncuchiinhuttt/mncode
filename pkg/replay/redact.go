package replay

import (
	"encoding/json"
	"reflect"
	"strings"

	"mncode/pkg/artifacts"
)

func marshalSafe(value any) (json.RawMessage, error) {
	if value == nil {
		return json.RawMessage("null"), nil
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var generic any
	if err := json.Unmarshal(encoded, &generic); err != nil {
		return nil, err
	}
	encoded, err = json.Marshal(scrubValue(generic, ""))
	if err != nil {
		return nil, err
	}
	return json.RawMessage(encoded), nil
}

func scrubValue(value any, key string) any {
	if sensitiveKey(key) {
		return "[REDACTED]"
	}
	switch item := value.(type) {
	case string:
		return artifacts.ScrubSecrets(item)
	case []any:
		out := make([]any, 0, len(item))
		for _, child := range item {
			out = append(out, scrubValue(child, key))
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(item))
		for childKey, child := range item {
			if !imageKey(childKey) {
				out[childKey] = scrubValue(child, childKey)
			}
		}
		return out
	default:
		return scrubReflect(value, key)
	}
}

func scrubReflect(value any, key string) any {
	rv := reflect.ValueOf(value)
	if !rv.IsValid() {
		return nil
	}
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return nil
		}
		return scrubReflect(rv.Elem().Interface(), key)
	}
	return value
}
func sensitiveKey(key string) bool {
	key = strings.ToLower(strings.NewReplacer("_", "", "-", "", " ", "").Replace(key))
	for _, marker := range []string{"apikey", "authorization", "password", "secret", "token", "privatekey", "thoughtsignature"} {
		if strings.Contains(key, marker) {
			return true
		}
	}
	return false
}
func imageKey(key string) bool {
	key = strings.ToLower(strings.NewReplacer("_", "", "-", "").Replace(key))
	return strings.Contains(key, "image") || strings.Contains(key, "inlinedata") || strings.Contains(key, "base64")
}
