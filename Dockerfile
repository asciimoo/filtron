# STEP 1 build executable binary
FROM golang:alpine as builder

WORKDIR $GOPATH/src/github.com/asciimoo/filtron

RUN apk add --no-cache git

COPY . .
RUN go get -d -v
RUN go build .

# STEP 2 build the image including only the binary
FROM alpine:latest

EXPOSE 4004 4005

WORKDIR /
RUN apk --no-cache add ca-certificates
RUN mkdir /etc/filtron

ADD configs/rules.json /etc/filtron/rules.json

COPY --from=builder /go/src/github.com/asciimoo/filtron/filtron /usr/bin/filtron

ENTRYPOINT ["/usr/bin/filtron", "--rules", "/etc/filtron/rules.json"]
