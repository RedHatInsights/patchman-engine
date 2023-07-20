FROM docker.io/confluentinc/cp-kafka

ADD /dev/kafka/entrypoint.sh /app/entrypoint.sh
ADD /dev/kafka/setup.sh      /app/setup.sh
ADD /dev/kafka/secrets       /etc/kafka/secrets

CMD ["/app/entrypoint.sh"]
