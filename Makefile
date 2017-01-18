test:
	go test -v -parallel=4 .

run:
	go run rrdserver.go

deps:
	glide i
