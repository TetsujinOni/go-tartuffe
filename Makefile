.PHONY: build run test clean test-api

# Build the binary
build:
	go build -o bin/tartuffe ./cmd/tartuffe

# Run the server
run: build
	./bin/tartuffe

# Run with injection allowed (for testing)
run-test: build
	./bin/tartuffe --allowInjection --localOnly

# Clean build artifacts
clean:
	rm -rf bin/

# Run Go tests
test:
	go test ./...

# Run mountebank API tests against go-tartuffe
# Requires mountebank repo at ../mountebank
test-api: build
	@echo "Starting go-tartuffe..."
	./bin/tartuffe --allowInjection --localOnly &
	@sleep 2
	@echo "Running mountebank API tests..."
	cd ../mountebank && npm run test:api || true
	@echo "Stopping go-tartuffe..."
	pkill -f "bin/tartuffe" || true

# Format code
fmt:
	go fmt ./...

# Run linter
lint:
	go vet ./...
