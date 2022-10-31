.PHONY: default build
default: build
# Self-written part
build: node.go client.go
	 go build client.go values.go && go build node.go values.go
