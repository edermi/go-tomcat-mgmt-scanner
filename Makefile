TARGET=go-tomcat-mgmt-scanner

all: prepare build-jobs

build-jobs: build-x86-linux build-x64-linux build-armv7-linux build-x64-windows build-x86-windows build-darwin

prepare:
	mkdir -p ./build

build-x86-linux: 
	GOOS=linux GOARCH=386 go build -o build/$(TARGET)-x86-linux . 

build-x64-linux: 
	GOOS=linux GOARCH=amd64 go build -o build/$(TARGET)-x64-linux . 

# raspberry pi
build-armv7-linux:
	GOOS=linux GOARCH=arm GOARM=7 go build -o build/$(TARGET)-armv7-linux . 

build-x64-windows: 
	GOOS=windows GOARCH=amd64 go build -o build/$(TARGET)-x64-windows.exe . 

build-x86-windows: 
	GOOS=windows GOARCH=386 go build -o build/$(TARGET)-x86-windows.exe . 

build-darwin: 
	GOOS=darwin go build -o build/$(TARGET)-macos . 

calculate-hashes:
	sha256sum build/* > build/checksums.txt

release: all calculate-hashes 
