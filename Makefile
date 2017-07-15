all: build
build:
	go build -o bin/memdbfs ./...
run: build
	bash demo/run.sh
optest:
	bash demo/optest.sh
