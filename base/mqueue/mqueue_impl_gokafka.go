package mqueue

import (
	"app/base/utils"
	"context"
	"crypto/tls"
	"crypto/x509"
	"github.com/segmentio/kafka-go"
	"io/ioutil"
	"time"
)

type kafkaGoReaderImpl struct {
	kafka.Reader
}

type kafkaGoWriterImpl struct {
	*kafka.Writer
}

func (t *kafkaGoWriterImpl) WriteMessages(ctx context.Context, msgs ...KafkaMessage) error {
	kafkaGoMessages := make([]kafka.Message, len(msgs))
	for i, m := range msgs {
		kafkaGoMessages[i] = kafka.Message{Key: m.Key, Value: m.Value}
	}
	err := t.Writer.WriteMessages(ctx, kafkaGoMessages...)
	return err
}

func NewKafkaGoReaderFromEnv(topic string) Reader {
	kafkaAddress := utils.GetenvOrFail("KAFKA_ADDRESS")
	kafkaGroup := utils.GetenvOrFail("KAFKA_GROUP")
	minBytes := utils.GetIntEnvOrDefault("KAFKA_READER_MIN_BYTES", 1)
	maxBytes := utils.GetIntEnvOrDefault("KAFKA_READER_MAX_BYTES", 1e6)

	config := kafka.ReaderConfig{
		Brokers:     []string{kafkaAddress},
		Topic:       topic,
		GroupID:     kafkaGroup,
		MinBytes:    minBytes,
		MaxBytes:    maxBytes,
		ErrorLogger: kafka.LoggerFunc(createLoggerFunc(kafkaErrorReadCnt)),
		Dialer:      tryCreateSecuredDialerFromEnv(),
	}

	reader := &kafkaGoReaderImpl{*kafka.NewReader(config)}
	return reader
}

func NewKafkaGoWriterFromEnv(topic string) Writer {
	kafkaAddress := utils.GetenvOrFail("KAFKA_ADDRESS")

	config := kafka.WriterConfig{
		Brokers: []string{kafkaAddress},
		Topic:   topic,
		// By default the writer will wait for a second (or until the buffer is filled by different goroutines)
		// before sending the batch of messages. Disable this, and use it in 'non-batched' mode
		// meaning single messages are sent immediately for now. We'll maybe change this later if the
		// sending overhead is a bottleneck
		BatchTimeout: time.Nanosecond,
		ErrorLogger:  kafka.LoggerFunc(createLoggerFunc(kafkaErrorWriteCnt)),
		Dialer:       tryCreateSecuredDialerFromEnv(),
	}
	writer := &kafkaGoWriterImpl{kafka.NewWriter(config)}
	return writer
}

func NewKafkaGoWriter(kafkaAddress, topic string) Writer {
	kafkaGoWriter := kafka.NewWriter(kafka.WriterConfig{
		Brokers:  []string{kafkaAddress},
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	})
	writer := &kafkaGoWriterImpl{kafkaGoWriter}
	return writer
}

// Init encrypting dialer if env var configured or return nil
func tryCreateSecuredDialerFromEnv() *kafka.Dialer {
	enableKafkaSsl := utils.GetBoolEnvOrDefault("ENABLE_KAFKA_SSL", false)
	if !enableKafkaSsl {
		return nil
	}

	kafkaSslSkipVerify := utils.GetBoolEnvOrDefault("KAFKA_SSL_SKIP_VERIFY", false)
	tlsConfig := &tls.Config{InsecureSkipVerify: true} // nolint:gosec
	if !kafkaSslSkipVerify {
		tlsConfig = caCertTLSConfigFromEnv()
	}

	dialer := &kafka.Dialer{
		Timeout:   10 * time.Second,
		DualStack: true,
		TLS:       tlsConfig,
	}
	return dialer
}

func caCertTLSConfigFromEnv() *tls.Config {
	caCertPath := utils.GetenvOrFail("KAFKA_SSL_CERT")
	caCert, err := ioutil.ReadFile(caCertPath)
	if err != nil {
		panic(err)
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	tlsConfig := tls.Config{RootCAs: caCertPool} // nolint:gosec
	return &tlsConfig
}
