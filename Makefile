COMMENT= an experimental project that allows you connect to Azure Cloud Shell from a local terminal without a browser.
	
BINDIR=	/usr/local/sbin
UNAME:=	$(shell uname)

all: build-linux build-darwin build-windows

build-linux:
	GOARCH=amd64 GOOS=linux go build -o bin/linux/amd64/azshell .

build-darwin:
	GOARCH=amd64 GOOS=darwin go build -o bin/darwin/amd64/azshell .

build-windows:
	GOARCH=amd64 GOOS=windows go build -o bin/windows/amd64/azshell.exe .

install:
ifeq ($(UNAME), Linux)
	install -o root -g bin bin/linux/amd64/azshell $(BINDIR)
else ifeq ($(UNAME), Darwin)
	install -o root -g bin bin/darwin/amd64/azshell $(BINDIR)
endif

clean:
	rm -rf dist/
	rm -rf bin/
	rm azshell
