package rql

// ASTNode represents an AST node in the RQL. Marshal should return
// an interface{} value that works with ast.MarshalJSON. This would
// typically be either a map[string]interface{} (JSON object),
// an []interface{} (JSON array), or a primitive type like
// nil (null), float64 (number), string, boolean, and time.Time.
// Similarly, the input in Unmarshal is an interface{} value that
// was decoded by ast.UnmarshalJSON.
type ASTNode interface {
	Marshal() interface{}
	Unmarshal(interface{}) error
}
