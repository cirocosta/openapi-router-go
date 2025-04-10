.PHONY: build run clean openapi-gen

# variables
BINARY_NAME=openapi-router-go
BUILD_DIR=./build

# build the application
build:
	@echo "building ${BINARY_NAME}..."
	@mkdir -p ${BUILD_DIR}
	@go build -o ${BUILD_DIR}/${BINARY_NAME} ./cmd/openapi-router-go

# install dependencies
deps:
	@echo "downloading dependencies..."
	@go mod download

# run the application
run: build
	@echo "running ${BINARY_NAME}..."
	@${BUILD_DIR}/${BINARY_NAME} run

# generate OpenAPI documentation
openapi-gen: build
	@echo "generating OpenAPI documentation..."
	@${BUILD_DIR}/${BINARY_NAME} openapi-gen -o docs/openapi.json
	@echo "OpenAPI documentation generated at docs/openapi.json"

tidy:
	@go mod tidy


test:
	@go test -count=1 -v ./...
