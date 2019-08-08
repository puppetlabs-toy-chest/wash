// Package version reports Wash's version
package version

// BuildVersion reports Wash's build version. It is set with
// `go build -ldflags="-X github.com/puppetlabs/wash/cmd/version.BuildVersion=${VERSION}"`
// as part of tagged builds. A local build might use
// `version.BuildVersion=$(git describe --always)` instead.
var BuildVersion = "unknown"
