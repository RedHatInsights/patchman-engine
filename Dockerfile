ARG BUILDIMG=registry.access.redhat.com/ubi8
ARG RUNIMG=registry.access.redhat.com/ubi8-micro
FROM ${BUILDIMG} as buildimg

ARG INSTALL_TOOLS=no
ARG BUILD_TAGS=""

# install build, development and test environment
RUN FULL_RHEL=$(dnf repolist rhel-8-for-x86_64-baseos-rpms --enabled -q) ; \
    if [ -z "$FULL_RHEL" ] ; then \
        rpm -Uvh http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/centos-stream-repos-8-4.el8.noarch.rpm \
                 http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/centos-gpg-keys-8-4.el8.noarch.rpm && \
        sed -i 's/^\(enabled.*\)/\1\npriority=200/;' /etc/yum.repos.d/CentOS*.repo ; \
    fi

RUN dnf module -y enable postgresql:12 && \
    dnf install -y go-toolset-1.16.* postgresql git-core diffutils rpm-devel && \
    ln -s /usr/libexec/platform-python /usr/bin/python3

ENV GOPATH=/go \
    GO111MODULE=on \
    GOPROXY=https://proxy.golang.org \
    PATH=$PATH:/go/bin \
    BUILD_TAGS_ENV=$BUILD_TAGS \
    PKG_CONFIG_PATH=/usr/lib/pkgconfig:$PKG_CONFIG_PATH

# confluent-kafka-go is not built for aarch64
# https://github.com/confluentinc/confluent-kafka-go/issues/576#issuecomment-1009766473
RUN if [ "$(uname -m)" == "aarch64" ] ; then \
        # build librdkafka from source for aarch64
        git clone --depth 1 --branch v1.6.2 https://github.com/edenhill/librdkafka.git ; \
        pushd librdkafka ; \
        dnf install -y gcc-c++ make cyrus-sasl-devel cyrus-sasl-lib cyrus-sasl-gssapi libcurl-devel lz4-devel lz4-libs libtool ; \
        ./configure --install-deps ; \
        make ; \
        make install ; \
        ln -s /usr/local/lib/librdkafka.so.1 /usr/lib64/librdkafka.so.1 ; \
        ln -s /usr/local/lib/pkgconfig /usr/lib/pkgconfig ; \
        popd ; \
    fi

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
        | sh -s -- -b $(go env GOPATH)/bin v1.44.2 ; \
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

RUN go build ${BUILD_TAGS} -v main.go

# libs to be copied into runtime
RUN mkdir -p /go/lib64 && \
    ldd /go/src/app/main /bin/psql /lib64/libpq.so.5 \
    | awk '/=>/ {print $3}' \
    | sort -u \
    | while read lib ; do \
        ln -v -t /go/lib64/ -s $lib ; \
    done

EXPOSE 8080

# ---------------------------------------
# runtime image with only necessary stuff
FROM ${RUNIMG} as runtimeimg

# allows using shared librdkafka library
ENV BUILD_TAGS_ENV=""

# create insights user
RUN echo "insights:x:1000:0::/go:/bin/bash" >>/etc/passwd && \
    mkdir /go && \
    chown insights:root /go

# copy libs needed by main
COPY --from=buildimg /lib64/libpq.so.5 /go/lib64/* /lib64/

# copy postgresql binaries
COPY --from=buildimg /usr/bin/clusterdb /usr/bin/createdb /usr/bin/createuser \
                     /usr/bin/dropdb /usr/bin/dropuser /usr/bin/pg_dump \
                     /usr/bin/pg_dumpall /usr/bin/pg_isready /usr/bin/pg_restore \
                     /usr/bin/pg_upgrade /usr/bin/psql /usr/bin/reindexdb \
                     /usr/bin/vacuumdb /usr/bin/

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
