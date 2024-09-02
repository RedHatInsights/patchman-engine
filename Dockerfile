ARG BUILDIMG=registry.access.redhat.com/ubi8
ARG RUNIMG=registry.access.redhat.com/ubi8-micro
FROM ${BUILDIMG} as buildimg

ARG INSTALL_TOOLS=no

# install build, development and test environment
RUN dnf module enable -y postgresql:16 || curl -o /etc/yum.repos.d/postgresql.repo \
        https://copr.fedorainfracloud.org/coprs/mmraka/postgresql-16/repo/epel-8/mmraka-postgresql-16-epel-8.repo

RUN dnf install -y go-toolset postgresql diffutils rpm-devel pg_repack && \
    ln -s /usr/libexec/platform-python /usr/bin/python3

ENV GOPATH=/go \
    GO111MODULE=on \
    GOPROXY=https://proxy.golang.org \
    PATH=$PATH:/go/bin

# now add patchman sources and build app
RUN adduser -d /go --no-create-home insights
RUN mkdir -p /go/src/app && chown -R insights:insights /go
USER insights
WORKDIR /go/src/app

ADD --chown=insights:insights go.mod go.sum     /go/src/app/

RUN go mod download

RUN if [ "$INSTALL_TOOLS" == "yes" ] ; then \
        go install github.com/swaggo/swag/cmd/swag@v1.16.2 && \
        go install gotest.tools/gotestsum@v1.10.1 && \
        curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh \
        | sh -s -- -b $(go env GOPATH)/bin v1.55.2 ; \
    fi

ADD --chown=insights:insights dev/kafka/secrets/ca.crt /opt/kafka/
ADD --chown=insights:insights dev/database/secrets/pgca.crt /opt/postgresql/
ADD --chown=insights:insights dev/scripts              /go/src/app/dev/scripts
ADD --chown=insights:insights main.go                  /go/src/app/
ADD --chown=insights:insights turnpike                 /go/src/app/turnpike
ADD --chown=insights:insights platform                 /go/src/app/platform
ADD --chown=insights:insights scripts                  /go/src/app/scripts
ADD --chown=insights:insights database_admin           /go/src/app/database_admin
ADD --chown=insights:insights docs                     /go/src/app/docs
ADD --chown=insights:insights evaluator                /go/src/app/evaluator
ADD --chown=insights:insights listener                 /go/src/app/listener
ADD --chown=insights:insights tasks                    /go/src/app/tasks
ADD --chown=insights:insights base                     /go/src/app/base
ADD --chown=insights:insights manager                  /go/src/app/manager
ADD --chown=insights:insights VERSION                  /go/src/app/

RUN go build -v main.go

# libs to be copied into runtime
RUN mkdir -p /go/lib64 && \
    ldd /go/src/app/main /usr/bin/pg_repack \
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
RUN echo "insights:x:1000:1000::/go:/bin/bash" >>/etc/passwd && \
    echo "insights:x:1000:insights" >>/etc/group && \
    mkdir /go && \
    chown insights:insights /go

# copy root ca certs so we can access https://logs.us-east-1.amazonaws.com/
COPY --from=buildimg /etc/pki/tls/certs/ca-bundle.crt /etc/pki/tls/certs/

# FIPS dependencies
COPY --from=buildimg /etc/crypto-policies/ /etc/crypto-policies/
COPY --from=buildimg /usr/lib64/.lib* /usr/lib64/
COPY --from=buildimg /usr/lib64/libssl* /usr/lib64/
COPY --from=buildimg /usr/bin/pg_repack /usr/bin/

# copy libs needed by main
COPY --from=buildimg /go/lib64/* /lib64/

ADD --chown=insights:insights go.sum                     /go/src/app/
ADD --chown=insights:insights scripts                    /go/src/app/scripts
ADD --chown=insights:insights database_admin/*.sh        /go/src/app/database_admin/
ADD --chown=insights:insights database_admin/*.sql       /go/src/app/database_admin/
ADD --chown=insights:insights database_admin/schema      /go/src/app/database_admin/schema
ADD --chown=insights:insights database_admin/migrations  /go/src/app/database_admin/migrations
ADD --chown=insights:insights docs/v3/openapi.json       /go/src/app/docs/v3/
ADD --chown=insights:insights docs/admin/openapi.json    /go/src/app/docs/admin/
ADD --chown=insights:insights VERSION                    /go/src/app/

COPY --from=buildimg --chown=insights:insights /go/src/app/main /go/src/app/

USER insights
WORKDIR /go/src/app

EXPOSE 8080
