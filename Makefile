BUILD_DIR = bin
PROGRAM = kubebpfbox
BPF2GO_CC = clang

.PHONY: build clean run

all: build

build:
	go generate ./plugins/http/...
	go generate ./plugins/oomkill/...
	go generate ./plugins/tcpsynbl/...
	CGO_ENABLED=0 \
	go build -o $(BUILD_DIR)/$(PROGRAM) .

run:
	go generate plugins/http/
	CGO_ENABLED=0 \
	go build -o $(BUILD_DIR)/$(PROGRAM) .
	$(BUILD_DIR)/$(PROGRAM)

clean:
	rm -rf $(BUILD_DIR)
	rm -rf plugins/*/*bpfel_x86.go
	rm -rf plugins/*/*bpfel_x86.o

image:
	sh build/build.sh