#!/bin/bash

go run . -url=https://monzo.com -ext=jpg,jpeg,png,svg,gif,svg,mp3,pdf,js,css -paths=-deeplink,help/,blog/,legal/ -o=e2e/monzo_basic.out "$@"
