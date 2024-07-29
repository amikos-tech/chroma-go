generate:
	sh ./gen_api_v3.sh

build:
	go build -v ./...

.PHONY: test
test:
	go test -tags=basic -count=1 -v ./...

.PHONY: test-rf
test-rf:
	go test -tags=rf -count=1 -v ./...

.PHONY: test-ef
test-ef:
	go test -tags=ef -count=1 -v ./...

.PHONY: lint
lint:
	golangci-lint run

.PHONY: lint-fix
lint-fix:
	golangci-lint run --fix --skip-dirs=./swagger ./...

.PHONY: clean-lint-cache
clean-lint-cache:
	golangci-lint cache clean


.PHONY: server
server:
	sh ./scripts/chroma_server.sh
