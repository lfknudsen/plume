all: server client

server:
	go build -C cmd/plume; cd cmd/plume; install

client:
	go build -C cmd/upload; cd cmd/upload; install
