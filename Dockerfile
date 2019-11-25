FROM centos:8

RUN yum module -y install go-toolset postgresql && yum -y install git
ENV GOPATH=/go
ENV GO111MODULE=on

ADD go.mod  /go/src/app/
ADD go.sum  /go/src/app/

WORKDIR /go/src/app
RUN go mod vendor

ADD /base       /go/src/app/base
ADD /manager    /go/src/app/manager
ADD /listener   /go/src/app/listener
ADD /docs       /go/src/app/docs
ADD main.go     /go/src/app/

RUN adduser --gid 0 -d /go --no-create-home insights
RUN chown -R insights:0 /go
USER insights

ADD /scripts/*.sh /go/src/app/

RUN go build -v main.go

EXPOSE 8080
