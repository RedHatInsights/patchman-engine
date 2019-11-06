FROM golang:1.12.3

ADD go.mod /go/src/gin-container/
ADD go.mod /go/src/gin-container/

WORKDIR /go/src/gin-container

ENV GO111MODULE=on
RUN go mod vendor

ADD . /go/src/gin-container

RUN go build -v -o appbin

EXPOSE 8000

ENTRYPOINT ["/go/src/appbin"]
