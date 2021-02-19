FROM confluentinc/cp-enterprise-kafka:5.4.0

ADD /dev/kafka/entrypoint.sh /entrypoint.sh
ADD /dev/kafka/setup.sh      /setup.sh
ADD /dev/kafka/secrets       /etc/kafka/secrets

CMD ["/entrypoint.sh"]
