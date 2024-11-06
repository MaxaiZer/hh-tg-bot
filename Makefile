BINARY_NAME=app
ifeq ($(OS),Windows_NT)
    BINARY_NAME := $(BINARY_NAME).exe
endif

build:
	@echo "Building the application..."
	go build -o $(BINARY_NAME) ./cmd

clean:
	@echo "Cleaning the build..."
	rm -f $(BINARY_NAME)

test:
	@echo "Running tests..."
	go test -v ./...

run: build
	@echo "Running the application..."
	./$(BINARY_NAME)

.PHONY: build clean run test