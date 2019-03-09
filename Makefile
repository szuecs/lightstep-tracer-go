# tools
GO=go

default: build
.PHONY: default build test

build: 
	${GO} build github.com/lightstep/lightstep-tracer-go/...

test:
	${GO} test -v github.com/lightstep/lightstep-tracer-go/...
	${GO} test -race -v github.com/lightstep/lightstep-tracer-go/...

# When releasing significant changes, make sure to update the semantic
# version number in `./VERSION`, merge changes, then run `make release_tag`.
version.go: VERSION
	./tag_version.sh

release_tag:
	git tag -a v`cat ./VERSION`
	git push origin v`cat ./VERSION`
