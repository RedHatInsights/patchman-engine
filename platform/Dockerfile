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

ENV LC_ALL=en_US.utf8
ENV LANG=en_US.utf8
RUN pip install --upgrade pip pipenv kafka

EXPOSE 2181 9092

ADD /platform /platform/
ADD /data     /platform/data
ADD /scripts/* /platform/

CMD /platform/entrypoint.sh
