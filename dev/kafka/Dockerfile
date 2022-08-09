FROM confluentinc/cp-enterprise-kafka:7.0.5

ADD /dev/kafka/entrypoint.sh /app/entrypoint.sh
ADD /dev/kafka/setup.sh      /app/setup.sh
ADD /dev/kafka/secrets       /etc/kafka/secrets

CMD ["/app/entrypoint.sh"]
