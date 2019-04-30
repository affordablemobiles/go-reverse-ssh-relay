#!/bin/bash

go build -ldflags="-s -w" -o build/reverse-ssh-endpoint

upx -9 build/reverse-ssh-endpoint
