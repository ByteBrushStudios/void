all: build

build:
	@echo "Building VOID, please wait..."
	@go build -v

start: build
	@echo "Starting VOID in production, please wait..."
	@./void

dev:
	@echo "Starting VOID in development, please wait... "
	@go run main.go