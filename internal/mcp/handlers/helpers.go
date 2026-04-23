package handlers

// stringArg safely extracts a string argument from the args map.
func stringArg(args map[string]any, key string) (string, bool) {
	v, ok := args[key]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

// intArg safely extracts an int argument from the args map (JSON numbers come as float64).
func intArg(args map[string]any, key string) (int, bool) {
	v, ok := args[key]
	if !ok {
		return 0, false
	}
	switch n := v.(type) {
	case float64:
		return int(n), true
	case int:
		return n, true
	case int64:
		return int(n), true
	}
	return 0, false
}

// boolArg safely extracts a bool from args; returns false if not present.
func boolArg(args map[string]any, key string) bool {
	v, ok := args[key]
	if !ok {
		return false
	}
	b, ok := v.(bool)
	return ok && b
}

// stringSliceArg extracts a []string from an args map value (which may be []any).
func stringSliceArg(args map[string]any, key string) ([]string, bool) {
	v, ok := args[key]
	if !ok {
		return nil, false
	}
	switch arr := v.(type) {
	case []string:
		return arr, true
	case []any:
		result := make([]string, 0, len(arr))
		for _, item := range arr {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result, true
	}
	return nil, false
}

// stringSetArg extracts a set of strings from an args map value.
func stringSetArg(args map[string]any, key string) map[string]bool {
	values, ok := stringSliceArg(args, key)
	if !ok {
		return nil
	}
	set := make(map[string]bool, len(values))
	for _, value := range values {
		if value != "" {
			set[value] = true
		}
	}
	return set
}

// intSliceArg extracts a []int from an args map value (JSON arrays of numbers come as []any of float64).
func intSliceArg(args map[string]any, key string) ([]int, bool) {
	v, ok := args[key]
	if !ok {
		return nil, false
	}
	switch arr := v.(type) {
	case []int:
		return arr, true
	case []any:
		result := make([]int, 0, len(arr))
		for _, item := range arr {
			switch n := item.(type) {
			case float64:
				result = append(result, int(n))
			case int:
				result = append(result, n)
			}
		}
		return result, true
	}
	return nil, false
}

// containsString checks if a string slice contains a value.
func containsString(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}
