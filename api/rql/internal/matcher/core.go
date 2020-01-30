package matcher

// Matcher contains some useful helpers meant to make unmarshaling
// AST nodes easy. This is a bit hacky, but it DRY's up much of the
// unmarshaling code (for now).

type Matcher = func(interface{}) bool

func Array(firstElemMatcher Matcher) Matcher {
	return func(v interface{}) bool {
		array, ok := v.([]interface{})
		return ok && len(array) >= 1 && firstElemMatcher(array[0])
	}
}

func Value(v interface{}) Matcher {
	return func(v2 interface{}) bool {
		return v == v2
	}
}
