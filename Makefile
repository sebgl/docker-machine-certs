build:
	CGO_ENABLED=0 go build

build-linux:
	GOOS=linux CGO_ENABLED=0 go build