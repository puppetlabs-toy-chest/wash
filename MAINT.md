# Maintenance Task Documentation

## Licenses

Collecting licenses used by wash:
```
go get -v gopkg.in/src-d/go-license-detector.v2/..
go mod vendor
find . -name 'LICENSE*' -exec dirname {} \; | xargs license-detector
```
