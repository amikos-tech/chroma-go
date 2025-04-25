generate:
	echo "This is deprecated. 1.0 does not use generated client."
	sh ./gen_api_v3.sh

build:
	go build -v ./...

.PHONY: gotestsum-bin
gotestsum-bin:
	go install gotest.tools/gotestsum@latest

.PHONY: test
test: gotestsum-bin
	gotestsum \
		--format short-verbose \
		--rerun-fails=5 \
		--packages="./..." \
		--junitfile unit-v1.xml \
		-- \
		-v \
		-tags=basic \
		-coverprofile=coverage-v1.out \
		-timeout=30m

.PHONY: test-v2
test-v2: gotestsum-bin
	gotestsum \
        --format short-verbose \
        --rerun-fails=5 \
        --packages="./..." \
        --junitfile unit-v2.xml \
        -- \
        -v \
        -tags=basicv2 \
        -coverprofile=coverage-v2.out \
        -timeout=30m

.PHONY: test-rf
test-rf: gotestsum-bin
	gotestsum \
		--format short-verbose \
		--rerun-fails=5 \
		--packages="./..." \
		--junitfile unit-rf.xml \
		-- \
		-v \
		-tags=rf \
		-coverprofile=coverage-rf.out \
		-timeout=30m

.PHONY: test-ef
test-ef: gotestsum-bin
	gotestsum \
		--format short-verbose \
		--rerun-fails=5 \
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
	golangci-lint run --fix --skip-dirs=./swagger ./...

.PHONY: clean-lint-cache
clean-lint-cache:
	golangci-lint cache clean


.PHONY: server
server:
	sh ./scripts/chroma_server.sh
