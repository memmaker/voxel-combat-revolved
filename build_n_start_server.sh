#!/bin/zsh

go build -o ./bgserve -tags server . && ./bgserve
