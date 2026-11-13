app_name=supersand
build_dir=bin
go=go
cro=cargo 

all: build

build:
	$(go) build -o $(build_dir)/$(app_name) main.go
	

run: build
	./$(build_dir)/$(app_name)

run-sudo: build
	sudo ./$(build_dir)/$(app_name)
		

clean:
	rm -rf $(build_dir)
.PHONY: all build run run-sudo clean proto	build-rs

proto:
	mkdir -p model
	protoc \
		-I proto \
		--go_out=model \
		--go_opt=paths=source_relative \
		proto/*.proto

build-rs:
	cd ./zygote	&&	$(cro)	build	
	cp	zygote/target/debug/zygote	./$(build_dir)




