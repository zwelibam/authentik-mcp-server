.PHONY: generate build run smoke-test docker-build lint

generate:
	go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
	oapi-codegen --config oapi-codegen.yaml /tmp/authentik-schema.json

build:
	mkdir -p bin
	go build -o bin/authentik-mcp ./cmd/authentik-mcp/

run: build
	./bin/authentik-mcp

smoke-test: build
	./bin/authentik-mcp --smoke-test

docker-build:
	docker build -t authentik-mcp:latest .

lint:
	go vet ./...
