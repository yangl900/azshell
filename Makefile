all: build-linux build-darwin build-windows

build-linux:
	GOARCH=amd64 GOOS=linux go build -o bin/linux/amd64/azshell .

build-darwin:
	GOARCH=amd64 GOOS=darwin go build -o bin/darwin/amd64/azshell .

build-windows:
	GOARCH=amd64 GOOS=windows go build -o bin/windows/amd64/azshell.exe .

clean:
	rm -rf dist/
	rm -rf bin/
	rm azshell