FROM golang:latest

ADD . /go/src/URLShortener
WORKDIR /go/src/URLShortener
RUN go env -w GO111MODULE=auto
RUN go get URLShortener
RUN go build -o main .
ENTRYPOINT ["/go/src/URLShortener/main"]