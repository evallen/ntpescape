# Makefile

.ONESHELL: build
.SILENT: build

build:
	mkdir -p ./bin
	export KEY=$$(od -vN 16 -An -tx1 /dev/urandom | tr -d " \n")
	go build -o ./bin/send \
		-ldflags="-X \
		'github.com/evallen/ntpescape/common.KeyString=$${KEY}'" \
		./cmd/send/main.go
	go build -o ./bin/recv \
		-ldflags="-X \
		'github.com/evallen/ntpescape/common.KeyString=$${KEY}'" \
		./cmd/recv/main.go
	echo "Executables created with key: $${KEY}"
	echo "Placed at: ./bin/send ./bin/recv"

clean:
	rm ./bin/send
	rm ./bin/recv