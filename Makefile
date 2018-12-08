.PHONY: all build gen get tool-deps get-dep get-protoc dep-check clean java

idls := idl/health.proto idl/service.proto

all: install

install: build
	go install $$(go list ./... | grep -v /vendor/)

build: gen
	go build ./...

get:
	dep ensure

gen: $(idls) 
	mkdir -p build/gen
	protoc -I=idl --go_out=plugins=grpc:build/gen idl/*.proto 

tool-deps: get-dep get-protoc

get-dep: 
	# install dep to manage the dependencies
	curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh

get-protoc:
	# the latest protoc-gen-go breaks protobuf v1.2.0 
	# let's use the vendored version for now
	go install ./vendor/github.com/golang/protobuf/protoc-gen-go

dep-check:
	@echo "=> checking dependencies"
	dep check

clean:
	go clean ./...

java:
	cd client/java && mvn package
