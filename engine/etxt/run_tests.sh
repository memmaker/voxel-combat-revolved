#!/bin/bash

# Notice: requires a test_font.ttf file in etxt/
#         (any normal font will work)
go test -tags gtxt ./... -count=1 -cover | grep "^[^?]"
