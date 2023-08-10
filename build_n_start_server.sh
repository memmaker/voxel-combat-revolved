#!/bin/zsh

## build linux version
go build -o ./bgserve -tags server . && ./bgserve

