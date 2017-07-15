all: build
build:
	go build -o bin/memdbfs ./...
deps:
	go get -u bazil.org/fuse
	go get -u github.com/hashicorp/go-memdb
	go get -u github.com/hashicorp/logutils
run: build
	bash demo/run.sh
optest:
	bash demo/optest.sh
