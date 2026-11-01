# SPDX-License-Identifier: GPL-3.0-or-later

BIN := bin/situ

.PHONY: build test vet fmt check clean

build:
	CGO_ENABLED=0 go build -trimpath -o $(BIN) ./cmd/situ

test:
	go test ./...

vet:
	go vet ./...

fmt:
	@out=$$(gofmt -l .); \
	if [ -n "$$out" ]; then \
		printf 'gofmt needed on:\n%s\n' "$$out" >&2; \
		exit 1; \
	fi

check: fmt vet test

clean:
	rm -rf bin
