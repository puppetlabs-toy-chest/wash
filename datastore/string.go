package datastore

import (
	"fmt"
	"path"
	"sort"
	"strings"
)

// ContainsString returns whether the named string is included in strings,
// assuming that the array of strings is sorted.
func ContainsString(names []string, name string) bool {
	idx := sort.SearchStrings(names, name)
	return idx < len(names) && names[idx] == name
}

// FindCompositeString returns whether the name is present in the array of sorted composite
// strings. Composite strings are token1/token2, where name is matched against token1.
func FindCompositeString(names []string, name string) (string, bool) {
	idx := sort.Search(len(names), func(i int) bool {
		x, _ := SplitCompositeString(names[i])
		return x >= name
	})
	if idx < len(names) {
		x, _ := SplitCompositeString(names[idx])
		if x == name {
			return names[idx], true
		}
	}
	return "", false
}

// SplitCompositeString splits a string around a '/' separator, requiring that the string only have
// a single separator.
func SplitCompositeString(id string) (string, string) {
	tokens := strings.Split(id, "/")
	if len(tokens) != 2 {
		panic(fmt.Sprintf("SplitCompositeString given an invalid name/id pair: %v", id))
	}
	return tokens[0], tokens[1]
}

// MakeCompositeString makes a composite string by joining name and extra with a '/' separator.
func MakeCompositeString(name string, extra string) string {
	return name + "/" + extra
}

// Keys returns an array of keys consisting of all '/' separated subkeys between base and start,
// including an optional suffix.
func Keys(base string, start string, suffix string) []string {
	ks := make([]string, 0)
	for pre := start; pre != "" && pre != "/"; pre = path.Dir(pre) {
		ks = append(ks, base+pre+suffix)
	}
	return append(ks, base+suffix)
}
