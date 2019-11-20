FROM centos:8

RUN yum module -y install go-toolset postgresql && yum -y install git
ENV GOPATH=/go

ADD go.mod  /go/src/app/
ADD main.go /go/src/app/

WORKDIR /go/src/app

ENV GO111MODULE=on

ADD /base       /go/src/app/base
ADD /webserver  /go/src/app/webserver
ADD /listener   /go/src/app/listener

RUN adduser --gid 0 -d /go --no-create-home insights
RUN chown -R insights:0 /go
USER insights

RUN go mod vendor
RUN go build -v main.go

EXPOSE 8080
