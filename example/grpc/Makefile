
GOPATH ?= `go env GOPATH`

gen:
	go get -d -v github.com/golang/protobuf/protoc-gen-go
	PATH="${PATH}:$(GOPATH)/bin" protoc -I=. -I=$(GOPATH)/src \
		--go_out=plugins=grpc:. server/*.proto
