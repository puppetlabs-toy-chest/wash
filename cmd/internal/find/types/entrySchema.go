package types

import (
	"sort"

	apitypes "github.com/puppetlabs/wash/api/types"
)

// EntrySchema is a wrapper to apitypes.EntrySchema
type EntrySchema struct {
	// "nil" means an unknown schema
	*apitypes.EntrySchema
	// Use a map for faster lookup
	Children map[string]*EntrySchema
}

// NewEntrySchema returns a new EntrySchema object. The returned
// object is faithful to s' representation as a graph. This means
// that all duplicate nodes are filtered out and that cycles are
// collapsed.
func NewEntrySchema(s *apitypes.EntrySchema) *EntrySchema {
	nodes := make(map[string]*EntrySchema)

	var gatherNodes func(s *apitypes.EntrySchema)
	gatherNodes = func(s *apitypes.EntrySchema) {
		if len(s.TypeID) == 0 {
			panic("gatherNodes called with an empty type ID!")
		}
		node := nodes[s.TypeID]
		if node == nil {
			node = &EntrySchema{
				EntrySchema: s,
				Children:    make(map[string]*EntrySchema),
			}
			nodes[s.TypeID] = node
		}
		if len(node.Children) > 0 {
			// We already gathered everything we need to know about this node
			return
		}
		for _, child := range s.Children {
			// The node's children will be filled in after all the nodes have been
			// gathered.
			node.Children[child.TypeID] = nil
			gatherNodes(child)
		}
	}
	gatherNodes(s)

	for _, node := range nodes {
		for childTypeID := range node.Children {
			node.Children[childTypeID] = nodes[childTypeID]
		}
	}
	return nodes[s.TypeID]
}

// Prune prunes s to contain only nodes that satisfy p. It modifies s'
// state so it is not an idempotent operation.
func Prune(s *EntrySchema, p EntrySchemaPredicate) *EntrySchema {
	keep := evaluateNodesToKeep(s, p)
	if !keep[s.TypeID] {
		return nil
	}

	visited := make(map[string]bool)
	var prune func(s *EntrySchema)
	prune = func(s *EntrySchema) {
		if visited[s.TypeID] {
			return
		}
		visited[s.TypeID] = true
		for _, child := range s.Children {
			if !keep[child.TypeID] {
				delete(s.Children, child.TypeID)
			} else {
				prune(child)
			}
		}
	}
	prune(s)

	return s
}

/*
"result" represents the returned value, which is a map of <node> => <keep?>.
Thus, result[N] == false means we prune N, result[N] == true means we keep
it. result[N] == true if:
	* p(N) returns true
	* p(N) returns true for at least one of N's children (ignoring self-loops)
Since entry schemas can contain cycles (e.g. the volume.dir class in the
volume package), and since they do not strictly adhere to a tree structure
(e.g. paths A-B-C and A-D-C are possible), we cannot completely calculate
"result" in a single iteration. For example in the A-B-C, A-D-C case, assume
that p returns false for A, B, C and D. Then we keep B and D if we keep C,
and we keep C if we keep any of its children. If C does not contain a cycle,
then we can recurse back the information to B and D. If C does contain a cycle,
say back to B, then it is difficult, if not impossible, to coordinate all the
updates in a single iteration. However if we let "result[N] == p(N) for all N
such that either p(N) == true or N has no children" be our starting state, then
we can iteratively evaluate result[V] for all other nodes V using the information
provided by a previous iteration. The terminating condition is when an iteration
does not update "result". This is possible if "result" contains all nodes N, or if
there are some indeterminate nodes V. In the latter case, every node M in V is either
part of a cycle, or there is more than one way to get to M from the root. In either
case, we can say that result[M] == false for those nodes.

NOTE: To see why result[M] == false for all M in V, we do a proof by contradiction.
Assume result[M] == true for some M in V, and that M is part of cycle A-B-...-M...-C-A.
Here, we see that one iteration would update all ancestors of M. A subsequent iteration
would update M's descendants starting from C. Thus, all the nodes in the cycle are
determinant, which contradicts our previous assumption of their indeterminacy (since
those nodes are also part of V). The proof for the other case is similar.

NOTE: The starting state formalizes the first condition for result[N] (and also notes
that if N is a leaf, then result[N] == p(N)). Subsequent iterations check the second
condition.
*/
func evaluateNodesToKeep(s *EntrySchema, p EntrySchemaPredicate) map[string]bool {
	result := make(map[string]bool)
	visited := make(map[string]bool)

	// Set-up our starting state by evaluating p(N). Note that after this code,
	// result[N] == p(N) for all nodes N such that p(N) == true or N has no
	// children.
	var applyPredicate func(s *EntrySchema)
	applyPredicate = func(s *EntrySchema) {
		if _, ok := visited[s.TypeID]; ok {
			return
		}
		result[s.TypeID] = p(s)
		if !result[s.TypeID] && len(s.Children) > 0 {
			delete(result, s.TypeID)
		}
		visited[s.TypeID] = true
		for _, child := range s.Children {
			applyPredicate(child)
		}
	}
	applyPredicate(s)

	// Now we iteratively update "result".
	var updateResult func(s *EntrySchema)
	updateResult = func(s *EntrySchema) {
		if _, ok := visited[s.TypeID]; ok {
			return
		}
		visited[s.TypeID] = true
		for _, child := range s.Children {
			updateResult(child)
			if result[child.TypeID] {
				result[s.TypeID] = true
			}
		}
	}
	for {
		visited = make(map[string]bool)
		prevNumNodes := len(result)
		updateResult(s)
		newNodes := len(result) - prevNumNodes
		if newNodes <= 0 {
			// We've reached the terminating condition. Note that result[V] == false
			// for an indeterminate node V since false is the 0-value for a "bool" type.
			// Thus, we can just return here.
			return result
		}
	}
}

// toMap returns a map of <typeID> => <childTypeIDs...> (i.e. its pre-serialized
// graph representation). It is useful for testing.
func (s *EntrySchema) toMap() map[string][]string {
	mp := make(map[string][]string)
	var visit func(s *EntrySchema)
	visit = func(s *EntrySchema) {
		if _, ok := mp[s.TypeID]; ok {
			return
		}
		mp[s.TypeID] = []string{}
		for _, child := range s.Children {
			mp[s.TypeID] = append(mp[s.TypeID], child.TypeID)
			visit(child)
		}
		sort.Strings(mp[s.TypeID])
	}
	visit(s)
	return mp
}
