FROM golang:1.12.3

ADD go.mod  /go/src/gin-container/
ADD go.mod  /go/src/gin-container/
ADD main.go /go/src/gin-container/

WORKDIR /go/src/gin-container

ENV GO111MODULE=on
RUN go mod vendor

ADD ./app /go/src/gin-container/app

EXPOSE 8000
