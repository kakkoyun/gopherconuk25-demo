.PHONY: all clean build toolexec loginjector demo demo-logging help

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
	go build -a -o $(BIN_DIR)/loginjector ./cmd/loginjector
	go build -a -o $(BIN_DIR)/main .

# Build just the toolexec wrapper
toolexec: $(BIN_DIR)
	go build -o $(BIN_DIR)/toolexecwrapper ./cmd/toolexecwrapper

# Build just the log injector
loginjector: $(BIN_DIR)
	go build -o $(BIN_DIR)/loginjector ./cmd/loginjector

demo: demo-toolexec demo-logging

# Demo toolexec
demo-toolexec: build
	@echo "=== Regular build ==="
	time go build -a -o $(BIN_DIR)/demo-regular .
	@echo
	@echo "=== Build with toolexec wrapper ==="
	go build -a -toolexec=$$(pwd)/$(BIN_DIR)/toolexecwrapper -o $(BIN_DIR)/demo-wrapped .

# Demo log injection
demo-logging: loginjector
	@echo "=== Running log injection demo ==="
	@echo "Original main.go:"
	@head -n 10 main.go
	@echo "..."
	@echo
	@echo "Injecting logging into main.go..."
	./$(BIN_DIR)/loginjector main.go
	@echo
	@echo "Generated main.go.generated (first 25 lines):"
	@head -n 25 main.go.generated
	@echo "..."
	@echo
	@echo "Running original program:"
	go run main.go
	@echo
	@echo "Running logged program (compile from generated source):"
	cp main.go.generated main_temp.go && go run main_temp.go && rm main_temp.go

# Clean built binaries
clean:
	rm -rf $(BIN_DIR)
	rm -f main.go.generated main_temp.go

# Help target
help:
	@echo "Available targets:"
	@echo "  all          - Build all binaries (default)"
	@echo "  build        - Build all binaries"
	@echo "  toolexec     - Build only the toolexec wrapper"
	@echo "  loginjector  - Build only the log injector tool"
	@echo "  demo         - Run demo comparing regular vs wrapped builds"
	@echo "  demo-toolexec - Run demo comparing regular vs wrapped builds"
	@echo "  demo-logging - Demonstrate log injection functionality"
	@echo "  clean        - Remove all built binaries and generated files"
	@echo "  help         - Show this help message"
