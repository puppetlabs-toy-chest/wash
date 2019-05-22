// Package params represents `wash find`'s parameters. These are typically
// set in `wash find`'s main function.
package params

import "time"

// ReferenceTime is the reference time that's used for `wash find`'s
// time predicates. Defaults to `wash find`'s start time.
var ReferenceTime time.Time
