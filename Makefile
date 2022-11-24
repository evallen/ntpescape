# Makefile

.ONESHELL: build
.SILENT: build

build:
	mkdir -p ./bin
	if [ "$$RECVGOOS" = "windows" ]; then
		export RECVNAME="./bin/recv.exe"
	else
		export RECVNAME="./bin/recv"
	fi
	if [ "$$SENDGOOS" = "windows" ]; then
		export SENDNAME="./bin/send.exe"
	else
		export SENDNAME="./bin/send"
	fi
	export KEY=$$(od -vN 16 -An -tx1 /dev/urandom | tr -d " \n")
	GOOS=$$SENDGOOS GOARCH=$$SENDGOARCH go build -o $$SENDNAME \
		-ldflags="-X \
		'github.com/evallen/ntpescape/common.KeyString=$${KEY}'" \
		./cmd/send/main.go
	GOOS=$$RECVGOOS GOARCH=$$RECVGOARCH go build -o $$RECVNAME \
		-ldflags="-X \
		'github.com/evallen/ntpescape/common.KeyString=$${KEY}'" \
		./cmd/recv/main.go
	echo "Executables created with key: $${KEY}"
	echo "Placed at: $$SENDNAME $$RECVNAME"

build-recv-windows:
	RECVGOOS=windows RECVGOARCH=amd64 make build
	
build-send-windows:
	SENDGOOS=windows SENDGOARCH=amd64 make build
	
build-send-windows-recv-windows:
	RECVGOOS=windows RECVGOARCH=amd64 make build SENDGOOS=windows SENDGOARCH=amd64 make build

clean:
	rm ./bin/*
