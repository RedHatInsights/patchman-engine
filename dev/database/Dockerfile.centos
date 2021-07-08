FROM centos/postgresql-12-centos8

ADD /database/init.sh ${CONTAINER_SCRIPTS_PATH}/start/

USER postgres

# copy custom config to enable SSL connections
ADD /dev/database/custom.conf /opt/app-root/src/postgresql-cfg/
ADD --chown=postgres:postgres /database/pg.* /opt/app-root/src/certificates/
RUN chmod 0600 /opt/app-root/src/certificates/pg.key

# copy config to enforce SSL connections to ensure all clients use SSL
ADD /dev/database/pg_hba.conf /opt/app-root/
