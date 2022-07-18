#!/usr/bin/env bash

[[ -n "${RKV_PID-}" ]] && sudo kill "${RKV_PID[@]}" 2>/dev/null

go run cmd/http/main.go
export RKV_PID=$!

