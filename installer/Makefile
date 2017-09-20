
# It's necessary to set this because some environments don't link sh -> bash.
SHELL := /bin/bash

OUT_DIR ?=output
BIN_DIR := $(OUT_DIR)/bin

all: generated_files build

generated_files:
	@go-bindata -pkg "main" -o cmd/templates.go templates/

build:
	@mkdir -p $(BIN_DIR) && go build -o $(BIN_DIR)/installer cmd/*.go

clean:
	@rm -rf $(OUT_DIR)

