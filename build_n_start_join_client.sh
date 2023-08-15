#!/bin/zsh

go build -o ./bgclient -tags client . && ./bgclient join "127.0.0.1:9999"

## compile for linux

