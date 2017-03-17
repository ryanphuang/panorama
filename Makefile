.PHONY: all idl

idls := idl/health.proto idl/service.proto
src := 

all: 
	go install ./...

idl:
	protoc -I=idl --go_out=plugins=grpc:build/gen idl/*.proto 
