# Makefile

build:
	mkdir -p ./bin
	go build -o ./bin/send ./cmd/send/main.go
	go build -o ./bin/recv ./cmd/recv/main.go

clean:
	rm ./bin/send
	rm ./bin/recv