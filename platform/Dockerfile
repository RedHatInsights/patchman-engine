FROM centos:7

RUN yum -y install centos-release-scl && \
    yum -y install rh-python36 which java-openjdk-headless && \
    rm -rf /var/cache/yum/*

ENV PATH=/opt/rh/rh-python36/root/usr/bin:$PATH


WORKDIR /platform

RUN cd /platform && \
    mkdir kafka && \
    curl https://www.apache.org/dist/kafka/2.2.0/kafka_2.12-2.2.0.tgz \
    | tar -xz --strip-components=1 -C kafka/

RUN adduser --gid 0 -d /platform --no-create-home insights && \
    chown -R insights /platform

USER insights

EXPOSE 9092

ADD /scripts/*.sh /platform/

CMD /platform/wait-for-kafka.sh
