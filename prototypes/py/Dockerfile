FROM centos:7

RUN yum -y install centos-release-scl && \
    yum -y install rh-python36 which postgresql postgresql-libs gcc && \
    rm -rf /var/cache/yum/*

ENV PATH=/opt/rh/rh-python36/root/usr/bin:$PATH

WORKDIR /prototype

ADD prototypes/py /prototype/
ADD scripts/* /prototype/

ENV LC_ALL=en_US.utf8
ENV LANG=en_US.utf8
RUN pip install --upgrade pip pipenv && \
    pipenv install --system

RUN adduser --gid 0 -d /prototype --no-create-home insights
USER insights

EXPOSE 8000

CMD /prototype/entrypoint.sh
