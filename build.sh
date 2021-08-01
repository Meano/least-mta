#!/bin/bash

go get

app="least-mta"

GOOS=windows go build -x -v -o release/windows/$app.exe
tar -zcvf release/$app-windows-x64.tar.gz release/windows/$app.exe

GOOS=linux go build -x -v -o release/linux/$app
tar -zcvf release/$app-linux-x64.tar.gz release/linux/$app
