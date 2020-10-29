# golang image
FROM golang:alpine

# install build tools
RUN apk --no-cache add rrdtool rrdtool-dev git gcc make build-base dropbear-scp

# copy your app into the image
WORKDIR /app
ADD . /app

# grab some dependencies
RUN go get -u github.com/gocarina/gocsv
RUN go get github.com/mattn/go-zglob
RUN go get github.com/ziutek/rrd

# build the app
RUN make build

# cp the built binary to bin
RUN cp rrdserver /bin/

# copy sample data to /data
# this should become configurable
RUN mkdir -p /data/sample
ADD ./sample /data/sample

# open port 9000
# this should become configurable
EXPOSE 9000

# start the server
CMD ["rrdserver", "-r", "/data"]
