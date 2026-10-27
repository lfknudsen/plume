all: server client

server:
	go build -C cmd/plume -o ../..; install

client:
	go build -C cmd/upload -o ../..; install
