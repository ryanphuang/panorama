.PHONY: all idl get java

idls := idl/health.proto idl/service.proto

all: idl get
	go install ./...

get: idl
	go get ./...

idl: $(idls) 
	mkdir -p build/gen
	protoc -I=idl --go_out=plugins=grpc:build/gen idl/*.proto 

clean:
	go clean ./...

java:
	cd client/java && mvn package
