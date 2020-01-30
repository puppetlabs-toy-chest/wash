package predicate

type ComparisonOp string

const (
	LT   ComparisonOp = "<"
	LTE  ComparisonOp = "<="
	GT   ComparisonOp = ">"
	GTE  ComparisonOp = ">="
	EQL  ComparisonOp = "="
	NEQL ComparisonOp = "!="
)

var comparisonOpMap = map[ComparisonOp]bool{
	LT:   true,
	LTE:  true,
	GT:   true,
	GTE:  true,
	EQL:  true,
	NEQL: true,
}
