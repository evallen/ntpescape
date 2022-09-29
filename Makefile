# Makefile

build-send: 
	mkdir -p ./bin
	go build -o ./bin/send ./cmd/send/main.go

build-recv:
	mkdir -p ./bin
	go build -o ./bin/recv ./cmd/recv/main.go

send: ./bin/send
	./bin/send

recv: ./bin/recv
	./bin/recv

clean:
	rm ./bin/send
	rm ./bin/recv