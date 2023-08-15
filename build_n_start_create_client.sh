#!/bin/zsh

go build -o ./bgclient -tags client . && ./bgclient create "127.0.0.1:9999"

## compile for linux

