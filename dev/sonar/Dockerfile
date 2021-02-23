FROM centos:8

RUN yum install -y unzip java && \
    yum clean all

ENV SONAR_VERSION=4.6.0.2311

RUN curl https://binaries.sonarsource.com/Distribution/sonar-scanner-cli/sonar-scanner-cli-${SONAR_VERSION}-linux.zip \
      -o sonar-scanner-cli-${SONAR_VERSION}-linux.zip && \
    unzip sonar-scanner-cli-${SONAR_VERSION}-linux.zip

ADD run.sh /run.sh
