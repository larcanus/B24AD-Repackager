APP_NAME := b24ad-repackager
SRC_DIR := source
MAIN := $(SRC_DIR)/main.go
OUT := $(APP_NAME)

.PHONY: all build clean run

all: build

build:
	go build -o $(OUT) $(MAIN)

run: build
	./$(OUT)

clean:
	rm -f $(OUT)
