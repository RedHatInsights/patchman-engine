FROM quay.io/psegedy/cp-kafka:7.7.0

ADD /dev/kafka/entrypoint.sh /app/entrypoint.sh
ADD /dev/kafka/setup.sh      /app/setup.sh
ADD /dev/kafka/secrets       /etc/kafka/secrets

CMD ["/app/entrypoint.sh"]
