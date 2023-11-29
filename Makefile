.PHONY: all
all: test
.SILENT:local-libjq
PWD=$(shell pwd)

libjq/include/jq.h:
	./scripts/build-libjq-go.sh

build-libjq: libjq/include/jq.h scripts/build-libjq-go.sh

test: build-libjq
	CGO_ENABLED=1 CGO_CFLAGS="-I$(PWD)/libjq/include" CGO_LDFLAGS="-L$(PWD)/libjq/lib" go test ./...
build:
	CGO_ENABLED=1 CGO_CFLAGS="-I$(PWD)/libjq/include" CGO_LDFLAGS="-L$(PWD)/libjq/lib" go build -a -installsuffix cgo y2c.go
