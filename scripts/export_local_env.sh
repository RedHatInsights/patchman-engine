CONFIGS="conf/local.env"
GITROOT=$(git rev-parse --show-toplevel)

export $(grep -h '^[[:alpha:]]' $CONFIGS | xargs) 

export ACG_CONFIG=$GITROOT/$ACG_CONFIG
export KAFKA_SSL_CERT=$GITROOT/$KAFKA_SSL_CERT
[[ -n $DB_SSLROOTCERT ]] && export DB_SSLROOTCERT=$GITROOT/$DB_SSLROOTCERT

