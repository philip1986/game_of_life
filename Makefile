build:
	echo $(shell pwd)
	export GOPATH=$(shell pwd)
	coffee -o ./assets/js/ ./assets/coffee/
	go build game.go

run:
	./game
