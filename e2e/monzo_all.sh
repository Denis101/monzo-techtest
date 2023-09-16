#!/bin/bash

go run . -url=https://monzo.com -ext=jpg,jpeg,png,svg,gif,svg,mp3,pdf,js,css -o=e2e/monzo_all.out "$@"
