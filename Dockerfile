ARG BUILDIMG=registry.access.redhat.com/ubi8
ARG RUNIMG=registry.access.redhat.com/ubi8-minimal
FROM ${BUILDIMG} as buildimg

ARG INSTALL_TOOLS=no

# install build, development and test environment

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
        go get -u github.com/swaggo/swag/cmd/swag && \
        curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh \
        | sh -s -- -b $(go env GOPATH)/bin latest ; \
    fi

ADD --chown=insights:root dev/kafka/secrets/ca.crt /opt/kafka/
ADD --chown=insights:root dev/database/secrets/pgca.crt /opt/postgresql/
ADD --chown=insights:root base                     /go/src/app/base
ADD --chown=insights:root database_admin           /go/src/app/database_admin
ADD --chown=insights:root docs                     /go/src/app/docs
ADD --chown=insights:root evaluator                /go/src/app/evaluator
ADD --chown=insights:root listener                 /go/src/app/listener
ADD --chown=insights:root manager                  /go/src/app/manager
ADD --chown=insights:root platform                 /go/src/app/platform
ADD --chown=insights:root scripts                  /go/src/app/scripts
ADD --chown=insights:root vmaas_sync               /go/src/app/vmaas_sync
ADD --chown=insights:root main.go                   /go/src/app/

RUN go build -v main.go

EXPOSE 8080

# ---------------------------------------
# runtime image with only necessary stuff
FROM ${RUNIMG} as runtimeimg

RUN microdnf install -y libpq rpm-build-libs && \
    microdnf clean all

RUN adduser --gid 0 -d /go --no-create-home insights

# copy postgresql binaries
COPY --from=buildimg /usr/bin/clusterdb /usr/bin/createdb /usr/bin/createuser \
                     /usr/bin/dropdb /usr/bin/dropuser /usr/bin/pg_dump \
                     /usr/bin/pg_dumpall /usr/bin/pg_isready /usr/bin/pg_restore \
                     /usr/bin/pg_upgrade /usr/bin/psql /usr/bin/reindexdb \
                     /usr/bin/vacuumdb /usr/bin/

RUN curl -L -o /usr/bin/haberdasher \
    https://github.com/RedHatInsights/haberdasher/releases/latest/download/haberdasher_linux_amd64 && \
    chmod 755 /usr/bin/haberdasher

ADD --chown=insights:root go.sum                     /go/src/app/
ADD --chown=insights:root scripts                    /go/src/app/scripts
ADD --chown=insights:root database_admin/*.sh        /go/src/app/database_admin/
ADD --chown=insights:root database_admin/*.sql       /go/src/app/database_admin/
ADD --chown=insights:root database_admin/schema      /go/src/app/database_admin/schema
ADD --chown=insights:root database_admin/migrations  /go/src/app/database_admin/migrations
ADD --chown=insights:root docs/openapi.json          /go/src/app/docs/
ADD --chown=insights:root vmaas_sync/entrypoint.sh   /go/src/app/vmaas_sync/

COPY --from=buildimg /go/src/app/main /go/src/app/

USER insights
WORKDIR /go/src/app

EXPOSE 8080
