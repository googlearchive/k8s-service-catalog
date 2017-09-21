##################################################################
# Copyright 2017 Google Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
##################################################################

# It's necessary to set this because some environments don't link sh -> bash.
SHELL := /bin/bash

OUT_DIR ?=output
BIN_DIR := $(OUT_DIR)/bin
SC_INSTALLER_NAME :="sc"

all: generated_files build

generated_files:
	@go-bindata -pkg "main" -o cmd/templates.go templates/

build:
	@mkdir -p $(BIN_DIR) && go build -o $(BIN_DIR)/sc cmd/*.go

clean:
	@rm -rf $(OUT_DIR)

