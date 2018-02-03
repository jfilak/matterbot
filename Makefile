GO=go

all: matterbot

matterbot: matterbot.go
	$(GO) build *.go
