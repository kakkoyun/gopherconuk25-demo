.PHONY: all clean build toolexec demo help

# Default target
all: build

# Build directory
BIN_DIR := bin

# Create bin directory if it doesn't exist
$(BIN_DIR):
	mkdir -p $(BIN_DIR)

# Build all binaries
build: $(BIN_DIR)
	go build -a -o $(BIN_DIR)/toolexecwrapper ./cmd/toolexecwrapper
	go build -a -o $(BIN_DIR)/main .

# Build just the toolexec wrapper
toolexec: $(BIN_DIR)
	go build -o $(BIN_DIR)/toolexecwrapper ./cmd/toolexecwrapper

# Demo commands
demo: build
	@echo "=== Regular build ==="
	time go build -a -o $(BIN_DIR)/demo-regular .
	@echo
	@echo "=== Build with toolexec wrapper ==="
	go build -a -toolexec=$$(pwd)/$(BIN_DIR)/toolexecwrapper -o $(BIN_DIR)/demo-wrapped .

# Clean built binaries
clean:
	rm -rf $(BIN_DIR)

# Help target
help:
	@echo "Available targets:"
	@echo "  all      - Build all binaries (default)"
	@echo "  build    - Build all binaries"
	@echo "  toolexec - Build only the toolexec wrapper"
	@echo "  demo     - Run demo comparing regular vs wrapped builds"
	@echo "  clean    - Remove all built binaries"
	@echo "  help     - Show this help message"
