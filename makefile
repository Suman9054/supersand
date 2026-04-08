app_name=supersand
build_dir=bin
go=go

all: build

build:
	$(go) build -o $(build_dir)/$(app_name) main.go

run: build
	./$(build_dir)/$(app_name)

run-sudo: build
	sudo ./$(build_dir)/$(app_name)
		

clean:
	rm -rf $(build_dir)
.PHONY: all build run clean
