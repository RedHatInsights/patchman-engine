# Dockerfile uses postgresql 10 latest image
# with applied vmaas database schema
FROM centos/postgresql-10-centos7

ENV POSTGRESQL_USER 'spm_admin'
ENV POSTGRESQL_PASSWORD 'spm_admin_pwd'
ENV POSTGRESQL_DATABASE 'spm'

ADD /database/*.sql ${CONTAINER_SCRIPTS_PATH}/start/
ADD /database/*.sh ${CONTAINER_SCRIPTS_PATH}/start/
