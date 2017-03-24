.PHONY: all idl

idls := idl/health.proto idl/service.proto

all: idl
	go install ./...

idl: $(idls)
	mkdir -p builg/gen
	protoc -I=idl --go_out=plugins=grpc:build/gen idl/*.proto 
