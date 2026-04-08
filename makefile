app_name=supersand
build_dir=bin
go=go
goflags=-ldflags="-s -w"

all: build

build:
mkdir -p $(build_dir)
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(go) build $(goflags) -o $(build_dir)/$(app_name) main.go

run: build
sudo ./$(build_dir)/$(app_name)

clean:
rm -rf $(build_dir)
