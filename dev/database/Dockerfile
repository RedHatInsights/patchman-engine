FROM quay.io/cloudservices/postgresql-rds:16

# install pg_repack
USER root
RUN curl -o /etc/yum.repos.d/postgresql.repo \
        https://copr.fedorainfracloud.org/coprs/g/insights/postgresql-16/repo/epel-9/group_insights-postgresql-16-epel-9.repo
RUN dnf module disable -y postgresql
RUN dnf install -y pg_repack

ADD /dev/database/init.sh /docker-entrypoint-initdb.d/

USER postgres

# copy custom config to enable SSL connections
ADD /dev/database/custom.conf /opt/app-root/src/postgresql-cfg/
ADD --chown=postgres:postgres /dev/database/secrets/pg.* /opt/app-root/src/certificates/
RUN chmod 0600 /opt/app-root/src/certificates/pg.key

# copy config to enforce SSL connections to ensure all clients use SSL
ADD /dev/database/pg_hba.conf /opt/app-root/
