BINARY=tkm
OUT_DIR=bin
DIST_DIR=dist

ifeq ($(OS),Windows_NT)
	EXE=.exe
	MKDIR=powershell -NoProfile -Command "New-Item -ItemType Directory -Force $(OUT_DIR),$(DIST_DIR) | Out-Null"
	RM=powershell -NoProfile -Command "Remove-Item -Recurse -Force -ErrorAction SilentlyContinue"
else
	EXE=
	MKDIR=mkdir -p $(OUT_DIR) $(DIST_DIR)
	RM=rm -rf
endif

.PHONY: build dist run test clean

build:
	$(MKDIR)
	go build -o $(OUT_DIR)/$(BINARY)$(EXE) .

dist:
	$(MKDIR)
	GOOS=windows GOARCH=amd64 go build -o $(DIST_DIR)/tkm-windows-amd64.exe .
	GOOS=linux GOARCH=amd64 go build -o $(DIST_DIR)/tkm-linux-amd64 .

run:
	go run . .

test:
	go test ./...

clean:
	$(RM) $(OUT_DIR) $(DIST_DIR)
