generate:
	sh ./gen_api_v3.sh

build:
	go build -v ./...

.PHONY: test
test:
	go test -v ./...

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

.PHONY: build-wasm-client
build-wasm-client:
	GOOS=js GOARCH=wasm go build -tags=wasm -o wasm/chromago.wasm wasm/main.go