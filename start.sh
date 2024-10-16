#!/usr/bin/env sh
CGO_ENABLED=0 go build -o ./assassingame_server && ./assassingame_server serve assassingame.org
