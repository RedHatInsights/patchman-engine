#!/usr/bin/env bash

# Download and add CA certificate if provided
if [[ ! -z "$SONAR_CERT_URL" ]]
then
   curl $SONAR_CERT_URL -o /CA.crt
   /usr/lib/jvm/jre-1.8.0/bin/keytool \
      -keystore /CA.keystore \
      -import -alias CA \
      -file /CA.crt \
      -noprompt -storepass passwd
   export SONAR_SCANNER_OPTS='-Djavax.net.ssl.trustStore=/CA.keystore -Djavax.net.ssl.trustStorePassword=passwd'
fi

# Create SonarQube config file using env variables
echo -e "sonar.projectKey=$PROJECT_KEY" > /sonar-scanner-${SONAR_VERSION}-linux/conf/sonar-scanner.properties
echo -e "sonar.projectName=$PROJECT_NAME" >> /sonar-scanner-${SONAR_VERSION}-linux/conf/sonar-scanner.properties
echo -e "sonar.host.url=$SONAR_HOST_URL" >> /sonar-scanner-${SONAR_VERSION}-linux/conf/sonar-scanner.properties
echo -e "sonar.login=$SONAR_LOGIN" >> /sonar-scanner-${SONAR_VERSION}-linux/conf/sonar-scanner.properties

# Do code analysis in mounted folder
cd /usr/src
exec /sonar-scanner-${SONAR_VERSION}-linux/bin/sonar-scanner
