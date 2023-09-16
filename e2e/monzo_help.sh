#!/bin/bash

go run . -url=https://monzo.com/help -ext=jpg,jpeg,png,svg,gif,svg,mp3,pdf,js,css -paths=-deeplink,i/,blog/,legal/ -o=e2e/monzo_help.out "$@"
