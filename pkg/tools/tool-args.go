package tools

func numberArgument(args map[string]interface{}, key string, fallback int) int {
	if value, ok := numberArg(args[key]); ok {
		return value
	}
	return fallback
}

func numberArg(value interface{}) (int, bool) {
	switch n := value.(type) {
	case int:
		return n, true
	case int8:
		return int(n), true
	case int16:
		return int(n), true
	case int32:
		return int(n), true
	case int64:
		return int(n), true
	case uint:
		return int(n), true
	case uint8:
		return int(n), true
	case uint16:
		return int(n), true
	case uint32:
		return int(n), true
	case uint64:
		return int(n), true
	case float64:
		return int(n), n == float64(int(n))
	case float32:
		return int(n), n == float32(int(n))
	default:
		return 0, false
	}
}
