#!/usr/bin/env bash

sudo apt update
sudo apt install redis-server
sudo systemctl restart redis.service