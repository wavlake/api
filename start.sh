#!/bin/bash
# Load environment variables from .env file, ignoring comments and empty lines
export $(grep -v '^#' .env | grep -v '^$' | xargs) && go run cmd/server/main.go