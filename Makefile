all: build
build:
	go build -o bin/memdbfs ./...
deps:
	go get golang.org/x/net/context
	go get bazil.org/fuse
	go get github.com/hashicorp/go-memdb
	go get github.com/hashicorp/logutils
run: build
	bash demo/run.sh
optest:
	bash demo/optest.sh
