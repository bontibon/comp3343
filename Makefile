PROTOC-GEN-GO = $(GOPATH)/bin/protoc-gen-go

all: server client

server: server.go protocol/protocol.pb.go
	go build -o $@ $<

client: client.go protocol/protocol.pb.go
	go build -o $@ $<

protocol/protocol.pb.go: protocol/protocol.proto
	protoc --plugin=protoc-gen-go=$(PROTOC-GEN-GO) --go_out=. $<

clean:
	rm -f client server protocol/protocol.pb.go

.PHONY: all clean
