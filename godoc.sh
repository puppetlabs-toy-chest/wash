#!/usr/bin/env sh

mkdir -p /tmp/tmpgoroot/doc
rm -rf /tmp/tmpgopath/src/github.com/puppetlabs/wash
mkdir -p /tmp/tmpgopath/src/github.com/puppetlabs/wash
tar -c --exclude='.git' --exclude='tmp' . | tar -x -C /tmp/tmpgopath/src/github.com/puppetlabs/wash
echo "open http://localhost:6060/pkg/github.com/puppetlabs/wash\n"
GOROOT=/tmp/tmpgoroot/ GOPATH=/tmp/tmpgopath/ godoc -http=localhost:6060

