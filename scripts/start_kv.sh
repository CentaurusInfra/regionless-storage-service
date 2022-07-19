#!/usr/bin/env bash

[[ -n "${RKV_PID-}" ]] && sudo kill "${RKV_PID[@]}" 2>/dev/null

make
./main &
export RKV_PID=$!
