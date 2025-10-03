
build:
	go build -v ./...

.PHONY: gotestsum-bin
gotestsum-bin:
	go install gotest.tools/gotestsum@latest

.PHONY: test
test: gotestsum-bin
	gotestsum \
        --format short-verbose \
        --rerun-fails=1 \
        --packages="./..." \
        --junitfile unit.xml \
        -- \
        -v \
        -tags=basicv2 \
        -coverprofile=coverage.out \
        -timeout=30m

.PHONY: test-cloud
test-cloud: gotestsum-bin
	gotestsum \
        --format short-verbose \
        --rerun-fails=1 \
        --packages="./..." \
        --junitfile unit-cloud.xml \
        -- \
        -p=1 \
        -v \
        -tags=basicv2,cloud \
        -coverprofile=coverage-cloud.out \
        -timeout=30m

.PHONY: test-ef
test-ef: gotestsum-bin
	gotestsum \
		--format short-verbose \
		--rerun-fails=1 \
		--packages="./..." \
		--junitfile unit-ef.xml \
		-- \
		-v \
		-tags=ef \
		-coverprofile=coverage-ef.out \
		-timeout=30m

.PHONY: lint
lint:
	golangci-lint run

.PHONY: lint-fix
lint-fix:
	golangci-lint run --fix ./...

.PHONY: clean-lint-cache
clean-lint-cache:
	golangci-lint cache clean


.PHONY: server
server:
	sh ./scripts/chroma_server.sh
