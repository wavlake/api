#!/bin/bash
export $(cat .env | xargs) && go run cmd/server/main.go