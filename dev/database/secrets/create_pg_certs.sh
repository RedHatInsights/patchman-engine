#!/bin/bash

# CA
openssl req -new -newkey rsa:4096 -days 10000 -x509 -subj "/CN=PGCA" -keyout pgca.key -out pgca.crt -nodes

## pg server
#openssl req -new -newkey rsa:4096 -days 10000 -x509 -subj "/CN=postgres" -addext "subjectAltName = DNS:db,DNS:localhost" \
#        -keyout pg.key -out pg.crt -nodes

# csr - signing request
openssl req -new -newkey rsa:4096 -subj "/CN=postgres" -out pg.csr -keyout pg.key -nodes

# sign csr with ca
echo "subjectAltName = DNS:db,DNS:localhost" >>/tmp/san.cnf
openssl x509 -req -CA pgca.crt -CAkey pgca.key -in pg.csr -out pg.crt -days 10000 -CAcreateserial -extfile /tmp/san.cnf -text
