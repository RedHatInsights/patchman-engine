FROM centos:8

RUN rpm --import /etc/pki/rpm-gpg/RPM-GPG-KEY-centosofficial && \
    yum module -y install go-toolset postgresql && \
    yum -y install git java-1.8.0-openjdk-devel
ENV GOPATH=/go
ENV GO111MODULE=on

ADD go.mod  /go/src/app/
ADD go.sum  /go/src/app/

RUN adduser --gid 0 -d /go --no-create-home insights
RUN chown -R insights:0 /go
USER insights

WORKDIR /go/src/app
RUN go mod download

RUN mkdir kafka && \
    curl https://www.apache.org/dist/kafka/2.2.0/kafka_2.12-2.2.0.tgz \
    | tar -xz --strip-components=1 -C kafka/

ADD /base           /go/src/app/base
ADD /platform/      /go/src/app/
ADD /scripts/*.sh   /go/src/app/

RUN go build -v platform.go inventory.go vmaas.go

EXPOSE 9001 2181 9092

CMD /go/src/app/entrypoint.sh

