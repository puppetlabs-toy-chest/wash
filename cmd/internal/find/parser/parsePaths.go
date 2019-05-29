package parser

var defaultPath = "."

func parsePaths(args []string) ([]string, []string) {
	paths := []string{}
	for {
		if len(args) == 0 {
			break
		}
		arg := args[0]
		if len(arg) == 0 {
			args = args[1:]
			continue
		}
		if arg[0] == '-' || isPartOfExpression(arg) {
			// arg is an option or part of a `wash find` expression
			break
		}
		paths = append(paths, arg)
		args = args[1:]
	}
	if len(paths) == 0 {
		paths = []string{defaultPath}
	}
	return paths, args
}