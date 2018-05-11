test:
	go test -v -parallel=4 .

build:
	go build -o rrdserver -v rrdserver.go

run:
	go run rrdserver.go

deps:
	glide i
