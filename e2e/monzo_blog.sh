#!/bin/bash

go run . -url=https://monzo.com/blog -ext=jpg,jpeg,png,svg,gif,svg,mp3,pdf,js,css -paths=-deeplink,i/,help/,legal/  -o=e2e/monzo_blog.out "$@"
