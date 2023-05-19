.PHONY: default build clean

default: build

build: client node

client: client.go values.go
	go build -o client client.go values.go

node: node.go values.go
	go build -o node node.go values.go

clean:
	rm -f client node
