.PHONY: run dev docker build test coverage coverage-html

# Variables
ENTRY_POINT=cmd/http/main.go
COVERAGE_FILE=coverage.out
COVERAGE_HTML=coverage.html

run:
	go run $(ENTRY_POINT)

dev:
	go run -gcflags=all="-N -l" $(ENTRY_POINT)

docker:
	docker compose up --build -d

build:
	go build $(ENTRY_POINT)

test:
	go test -v ./...

coverage:
	go test ./... -coverprofile=$(COVERAGE_FILE)
	go tool cover -func=$(COVERAGE_FILE)

coverage-html: coverage
	go tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@echo "Coverage report: $(COVERAGE_HTML)"