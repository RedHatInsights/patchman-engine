#!/usr/bin/bash

# 1. Create own private Certificate Authority (CA)
openssl req -new -newkey rsa:4096 -days 10000 -x509 -subj "/CN=kafka" -keyout ca.key -out ca.crt -nodes

# 2. Create kafka server certificate and store in keystore
keytool -genkey -keystore kafka.broker.keystore.jks -validity 10000 -storepass confluent -keypass confluent -dname "CN=kafka" -storetype pkcs12
# verify certificate
echo confluent | keytool -list -v -keystore kafka.broker.keystore.jks

# 3. Create Certificate signed request (CSR)
keytool -keystore kafka.broker.keystore.jks -certreq -file cert-file -storepass confluent -keypass confluent

# 4. Get CSR Signed with the CA:
openssl x509 -req -CA ca.crt -CAkey ca.key -in cert-file -out cert-file-signed -days 10000 -CAcreateserial -passin pass:confluent
# verify certificate
echo confluent | keytool -printcert -v -file cert-file-signed

# 5. Import CA certificate in KeyStore:
keytool -keystore kafka.broker.keystore.jks -alias CARoot -import -file ca.crt -storepass confluent -keypass confluent -noprompt

# 6. Import Signed CSR In KeyStore:
keytool -keystore kafka.broker.keystore.jks -import -file cert-file-signed -storepass confluent -keypass confluent -noprompt

# 7. Import CA certificate In TrustStore:
keytool -keystore kafka.broker.truststore.jks -alias CARoot -import -file ca.crt -storepass confluent -keypass confluent -noprompt
