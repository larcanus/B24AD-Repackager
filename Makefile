APP_NAME := b24ad-repackager
SRC_DIR := source
MAIN := $(SRC_DIR)/main.go
OUT := $(APP_NAME)

LINUX_OUT := $(APP_NAME)-linux
WINDOWS_OUT := $(APP_NAME)-windows.exe
MACOS_OUT := $(APP_NAME)-macos
MACOS_ARM_OUT := $(APP_NAME)-macos-arm64

.PHONY: all build clean run linux windows macos macos-arm

all: linux windows macos macos-arm

build:
	go build -o $(OUT) $(MAIN)

linux:
	CC=x86_64-linux-musl-gcc CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o $(LINUX_OUT) $(MAIN)

windows:
	CC=x86_64-w64-mingw32-gcc CGO_ENABLED=1 GOOS=windows GOARCH=amd64 go build -o $(WINDOWS_OUT) $(MAIN)

macos:
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -o $(MACOS_OUT) $(MAIN)

macos-arm:
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -o $(MACOS_ARM_OUT) $(MAIN)

run: build
	./$(OUT)

clean:
	rm -f $(OUT) $(LINUX_OUT) $(WINDOWS_OUT) $(MACOS_OUT) $(MACOS_ARM_OUT)
