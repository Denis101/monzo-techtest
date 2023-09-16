#!/bin/bash

go run . -url=https://monzo.com/legal -ext=jpg,jpeg,png,svg,gif,svg,mp3,pdf,js,css -paths=-deeplink,i/,help/,blog/ -o=e2e/monzo_legal.out "$@"
