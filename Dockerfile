ARG BUILDIMG=registry.redhat.io/ubi8:latest
ARG RUNIMG=registry.redhat.io/ubi8/ubi-minimal:latest
FROM ${BUILDIMG} as buildimg

ARG INSTALL_TOOLS=no

USER root

# install build, development and test environment
RUN FULL_RHEL=$(dnf repolist rhel-8-for-x86_64-baseos-rpms --enabled -q) ; \
    if [ -z "$FULL_RHEL" ] ; then \
        rpm -Uvh http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/centos-stream-repos-8-4.el8.noarch.rpm \
                 http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/centos-gpg-keys-8-4.el8.noarch.rpm && \
        sed -i 's/^\(enabled.*\)/\1\npriority=200/;' /etc/yum.repos.d/CentOS*.repo ; \
    fi

RUN dnf module -y enable postgresql:12 && \
    dnf install -y go-toolset postgresql git-core diffutils rpm-devel && \
    ln -s /usr/libexec/platform-python /usr/bin/python3

ENV GOPATH=/go \
    GO111MODULE=on \
    GOPROXY=https://proxy.golang.org \
    PATH=$PATH:/go/bin

WORKDIR /go/src/app

COPY . .

RUN go get -d ./...

RUN if [ "$INSTALL_TOOLS" == "yes" ] ; then \
        go install github.com/swaggo/swag/cmd/swag@v1.8.7 && \
        curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh \
        | sh -s -- -b $(go env GOPATH)/bin v1.50.0 ; \
    fi

RUN go build -v main.go

# libs to be copied into runtime
RUN mkdir -p /go/lib64 && \

    ldd /go/src/app/main \
    | awk '/=>/ {print $3}' \
    | sort -u \
    | while read lib ; do \
        ln -v -t /go/lib64/ -s $lib ; \
    done

# ---------------------------------------
# runtime image with only necessary stuff
FROM ${RUNIMG} as runtimeimg

COPY --from=buildimg /go/src/app/main /go/src/app/

WORKDIR /go/src/app

COPY . .

EXPOSE 8080
