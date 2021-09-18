all: vet fmt build build-windows

test:
		go test ./...

vendor:
		go vet ./...

vet:
		go vet ./...

fmt:
		go list -f '{{.Dir}}' ./... | grep -v /vendor/ | xargs -L1 gofmt -l
		# test -z $$(go list -f '{{.Dir}}' ./... | grep -v /vendor/ | xargs -L1 gofmt -l)

lint:
		go list ./... | grep -v /vendor/ | xargs -L1 golint -set_exit_status

build: build-windows build-osx build-linux

build-windows:
		GOOS=windows GOARCH=amd64 go build -o bin/mendix-userlib-cleaner.windows ./cmd/mendix-userlib-cleaner

build-osx:
		GOOS=darwin GOARCH=amd64 go build -o bin/mendix-userlib-cleaner.osx ./cmd/mendix-userlib-cleaner

build-linux:
		GOOS=linux GOARCH=amd64 go build -o bin/mendix-userlib-cleaner.linux ./cmd/mendix-userlib-cleaner
