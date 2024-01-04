generate:
	sh ./gen_api_v3.sh

build:
	go build -v ./...

gotest:
	go test -v ./...

lint:
	golangci-lint run