#!/usr/bin/env sh
git pull origin master
CGO_ENABLED=0 go build -o ./assassingame_server && ./assassingame_server serve --dev
