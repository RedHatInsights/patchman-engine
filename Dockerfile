ARG BUILDIMG=registry.access.redhat.com/ubi8
ARG RUNIMG=registry.access.redhat.com/ubi8-micro
FROM ${BUILDIMG} as buildimg

ARG INSTALL_TOOLS=no

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

# now add patchman sources and build app
RUN adduser --gid 0 -d /go --no-create-home insights
RUN mkdir -p /go/src/app && chown -R insights:root /go
USER insights
WORKDIR /go/src/app

ADD --chown=insights:root go.mod go.sum     /go/src/app/

RUN go mod download

RUN if [ "$INSTALL_TOOLS" == "yes" ] ; then \
        go install github.com/swaggo/swag/cmd/swag@v1.8.7 && \
        curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh \
        | sh -s -- -b $(go env GOPATH)/bin v1.50.0 ; \
    fi

ADD --chown=insights:root dev/kafka/secrets/ca.crt /opt/kafka/
ADD --chown=insights:root dev/database/secrets/pgca.crt /opt/postgresql/
ADD --chown=insights:root dev/scripts              /go/src/app/dev/scripts
ADD --chown=insights:root main.go                  /go/src/app/
ADD --chown=insights:root turnpike                 /go/src/app/turnpike
ADD --chown=insights:root platform                 /go/src/app/platform
ADD --chown=insights:root scripts                  /go/src/app/scripts
ADD --chown=insights:root database_admin           /go/src/app/database_admin
ADD --chown=insights:root docs                     /go/src/app/docs
ADD --chown=insights:root evaluator                /go/src/app/evaluator
ADD --chown=insights:root listener                 /go/src/app/listener
ADD --chown=insights:root tasks                    /go/src/app/tasks
ADD --chown=insights:root base                     /go/src/app/base
ADD --chown=insights:root manager                  /go/src/app/manager
ADD --chown=insights:root VERSION                  /go/src/app/

RUN go build -v main.go

# libs to be copied into runtime
RUN mkdir -p /go/lib64 && \
    ldd /go/src/app/main \
    | awk '/=>/ {print $3}' \
    | sort -u \
    | while read lib ; do \
        ln -v -t /go/lib64/ -s $lib ; \
    done

EXPOSE 8080

# ---------------------------------------
# runtime image with only necessary stuff
FROM ${RUNIMG} as runtimeimg

# create insights user
RUN echo "insights:x:1000:0::/go:/bin/bash" >>/etc/passwd && \
    mkdir /go && \
    chown insights:root /go

# copy root ca certs so we can access https://logs.us-east-1.amazonaws.com/
COPY --from=buildimg /etc/pki/tls/certs/ca-bundle.crt /etc/pki/tls/certs/

# FIPS dependencies
COPY --from=buildimg /etc/crypto-policies/ /etc/crypto-policies/
COPY --from=buildimg /usr/lib64/.lib* /usr/lib64/
COPY --from=buildimg /usr/lib64/libssl* /usr/lib64/

# copy libs needed by main
COPY --from=buildimg /go/lib64/* /lib64/

ADD --chown=insights:root go.sum                     /go/src/app/
ADD --chown=insights:root scripts                    /go/src/app/scripts
ADD --chown=insights:root database_admin/*.sh        /go/src/app/database_admin/
ADD --chown=insights:root database_admin/*.sql       /go/src/app/database_admin/
ADD --chown=insights:root database_admin/schema      /go/src/app/database_admin/schema
ADD --chown=insights:root database_admin/migrations  /go/src/app/database_admin/migrations
ADD --chown=insights:root docs/v1/openapi.json       /go/src/app/docs/v1/
ADD --chown=insights:root docs/v2/openapi.json       /go/src/app/docs/v2/
ADD --chown=insights:root docs/v3/openapi.json       /go/src/app/docs/v3/
ADD --chown=insights:root docs/admin/openapi.json    /go/src/app/docs/admin/
ADD --chown=insights:root VERSION                    /go/src/app/

COPY --from=buildimg /go/src/app/main /go/src/app/

USER insights
WORKDIR /go/src/app

EXPOSE 8080
