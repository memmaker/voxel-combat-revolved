#!/bin/zsh

## build client version
go build -o ./bgclient -tags client . && ./bgclient

