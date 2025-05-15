EXAMPLES_DIR := examples
BIN_DIR := bin

EXAMPLES := http_client_example

.PHONY: all examples clean lint test

all: examples lint

lint:
	golangci-lint run ./...

test:
	go test ./...

examples: $(EXAMPLES:%=$(BIN_DIR)/%)

$(BIN_DIR)/http_client_example: $(EXAMPLES_DIR)/http_client_example.go
	mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/http_client_example $(EXAMPLES_DIR)/http_client_example.go

clean:
	rm -rf $(BIN_DIR)
