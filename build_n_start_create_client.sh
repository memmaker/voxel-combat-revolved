#!/bin/zsh

go build -o ./bgclient -tags client . && ./bgclient create

## compile for linux

