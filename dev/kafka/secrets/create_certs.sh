#!/usr/bin/bash -e

# 1. Create own private Certificate Authority (CA)
openssl req -new -newkey rsa:4096 -days 10000 -x509 -subj "/CN=CA" -keyout ca.key -out ca.crt -nodes

# 2. Create kafka server certificate and store in keystore
openssl req -new -newkey rsa:4096 -days 10000 -x509 -subj "/CN=kafka" -addext "subjectAltName = DNS:kafka,DNS:localhost" \
        -keyout kafka.key -out kafka.crt -nodes
openssl pkcs12 -export -in kafka.crt -inkey kafka.key -out kafka.p12 -password pass:confluent
rm -f kafka.broker.keystore.jks
keytool -importkeystore -destkeystore kafka.broker.keystore.jks -deststorepass confluent -destkeypass confluent -deststoretype pkcs12 -destalias mykey \
        -srcstorepass confluent -srckeystore kafka.p12 -srcstoretype pkcs12 -srcalias 1 -noprompt
# verify certificate
keytool -list -v -keystore kafka.broker.keystore.jks -storepass confluent

# 3. Create Certificate signed request (CSR)
keytool -keystore kafka.broker.keystore.jks -certreq -file kafka.csr -storepass confluent -keypass confluent

# 4. Get CSR Signed with the CA:
echo "subjectAltName = DNS:kafka,DNS:localhost" >>san.cnf
openssl x509 -req -CA ca.crt -CAkey ca.key -in kafka.csr -out kafka-signed.crt -days 10000 -CAcreateserial -passin pass:confluent -extfile san.cnf
# verify certificate
keytool -printcert -v -file kafka-signed.crt -storepass confluent

# 5. Import CA certificate in KeyStore:
keytool -keystore kafka.broker.keystore.jks -alias CARoot -import -file ca.crt -storepass confluent -keypass confluent -noprompt

# 6. Import Signed CSR In KeyStore:
keytool -keystore kafka.broker.keystore.jks -import -file kafka-signed.crt -storepass confluent -keypass confluent -noprompt

# 7. Import CA certificate In TrustStore:
rm -f kafka.broker.truststore.jks
keytool -keystore kafka.broker.truststore.jks -alias CARoot -import -file ca.crt -storepass confluent -keypass confluent -noprompt

rm -f ca.{key,srl} kafka.{crt,csr,key,p12} kafka-signed.crt san.cnf
