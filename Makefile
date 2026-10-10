# Original ESP32 (Xtensa) firmware. The image runs on the chip via TinyGo.
# Host darwin/linux/windows binaries cannot run this program, so build-all
# compiles the ESP32 image into dist/ and runs the host tests for the
# push client.

TARGET   ?= esp32-generic
BIN_NAME ?= esp32-ex1
DIST_DIR ?= dist

# Example: make flash PORT=/dev/cu.usbserial-110
PORT     ?=
PORT_ARG := $(if $(PORT),-port=$(PORT),)

-include .env

WIFI_SSID      ?=
WIFI_PASS      ?=
PUSH_URL       ?= http://192.168.31.102:9091
PUSH_JOB       ?= esp32-ex1
PUSH_INSTANCE  ?= esp32

# Single-quote each value so spaces and '!' in a passphrase survive the shell.
LDFLAGS := -ldflags "\
 -X 'main.ssid=$(WIFI_SSID)'\
 -X 'main.password=$(WIFI_PASS)'\
 -X 'main.pushURL=$(PUSH_URL)'\
 -X 'main.pushJob=$(PUSH_JOB)'\
 -X 'main.pushInstance=$(PUSH_INSTANCE)'"

.PHONY: all build build-all flash monitor size test tidy clean help check-config

help:
	@echo "Targets:"
	@echo "  build      - TinyGo image to $(DIST_DIR)/$(BIN_NAME).bin (target $(TARGET))"
	@echo "  build-all  - host tests + firmware image in $(DIST_DIR)/"
	@echo "  flash      - flash the image (set PORT= when more than one serial device is attached)"
	@echo "  monitor    - serial monitor at 115200"
	@echo "  test       - go test ./internal/..."
	@echo "  clean      - remove $(DIST_DIR)/"
	@echo ""
	@echo "WiFi and the gateway come from .env (see .env.example)."
	@echo "  SSID=$(WIFI_SSID)"
	@echo "  PUSH_URL=$(PUSH_URL)"
	@echo "  POST $(PUSH_URL)/metrics/job/$(PUSH_JOB)/instance/$(PUSH_INSTANCE) every 10s"

all: build

check-config:
	@if test -z "$(WIFI_SSID)"; then \
		echo "WARN: WIFI_SSID is empty. Copy .env.example to .env before flashing."; \
	fi

build: check-config
	@mkdir -p $(DIST_DIR)
	@tinygo build -target=$(TARGET) -size short $(LDFLAGS) -o $(DIST_DIR)/$(BIN_NAME).bin .

build-all: test build

flash: check-config
	@tinygo flash -target=$(TARGET) -size short $(LDFLAGS) $(PORT_ARG) .

monitor:
	tinygo monitor -target=$(TARGET) -baudrate=115200 $(PORT_ARG)

size: check-config
	@mkdir -p $(DIST_DIR)
	@tinygo build -target=$(TARGET) -size full $(LDFLAGS) -o $(DIST_DIR)/$(BIN_NAME).bin .

test:
	go test ./internal/...

tidy:
	go mod tidy

clean:
	rm -rf $(DIST_DIR)
