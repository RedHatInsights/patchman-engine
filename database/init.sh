#!/bin/bash

# allow to create users for patchman database admin user
psql -c "ALTER USER ${POSTGRESQL_USER} WITH CREATEROLE"
psql -c "ALTER USER ${POSTGRESQL_USER} WITH SUPERUSER"