FROM centos:7
RUN yum -y install python3 gcc g++ gcc-c++ make git postgresql postgresql-libs postgresql-devel libpq-devel \
    openssl-devel clang clang-devel \
    && rm -rf /var/cache/yum/*
RUN yum -y install /usr/lib/libpq.so && rm -rf /var/cache/yum/*
RUN curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | \
    sh -s -- --default-toolchain stable --profile minimal -y -q

ENV PATH="/root/.cargo/bin:${PATH}"

WORKDIR /prototype

RUN mkdir -p /cargo_home /out
ENV CARGO_HOME="/cargo_home"

ADD ./prototypes/rust/ /prototype/

CMD cargo run --release --target-dir /out
