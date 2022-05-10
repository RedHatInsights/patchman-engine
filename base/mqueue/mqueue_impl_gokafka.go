package mqueue

import (
	"app/base"
	"app/base/utils"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl"
	kafkaPlain "github.com/segmentio/kafka-go/sasl/plain"
	kafkaScram "github.com/segmentio/kafka-go/sasl/scram"
)

type kafkaGoReaderImpl struct {
	kafka.Reader
}

func (t *kafkaGoReaderImpl) HandleMessages(handler MessageHandler) {
	for {
		m, err := t.FetchMessage(base.Context)
		if err != nil {
			if err.Error() == errContextCanceled {
				break
			}
			utils.Log("err", err.Error()).Error("unable to read message from Kafka reader")
			panic(err)
		}
		// At this level, all errors are fatal
		kafkaMessage := KafkaMessage{Key: m.Key, Value: m.Value}
		if err = handler(kafkaMessage); err != nil {
			utils.Log("err", err.Error()).Panic("Handler failed")
		}
		err = t.CommitMessages(base.Context, m)
		if err != nil {
			if err.Error() == errContextCanceled {
				break
			}
			utils.Log("err", err.Error()).Error("unable to commit kafka message")
			panic(err)
		}
	}
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

func NewKafkaReaderFromEnv(topic string) Reader {
	kafkaAddress := utils.GetenvOrFail("KAFKA_ADDRESS")
	kafkaGroup := utils.GetenvOrFail("KAFKA_GROUP")
	minBytes := utils.GetIntEnvOrDefault("KAFKA_READER_MIN_BYTES", 1)
	maxBytes := utils.GetIntEnvOrDefault("KAFKA_READER_MAX_BYTES", 1e6)
	maxAttempts := utils.GetIntEnvOrDefault("KAFKA_READER_MAX_ATTEMPTS", 3)

	config := kafka.ReaderConfig{
		Brokers:     []string{kafkaAddress},
		Topic:       topic,
		GroupID:     kafkaGroup,
		MinBytes:    minBytes,
		MaxBytes:    maxBytes,
		ErrorLogger: kafka.LoggerFunc(createLoggerFunc(kafkaErrorReadCnt)),
		Dialer:      tryCreateSecuredDialerFromEnv(),
		MaxAttempts: maxAttempts,
	}

	reader := &kafkaGoReaderImpl{*kafka.NewReader(config)}
	return reader
}

func NewKafkaWriterFromEnv(topic string) Writer {
	kafkaAddress := utils.GetenvOrFail("KAFKA_ADDRESS")
	maxAttempts := utils.GetIntEnvOrDefault("KAFKA_WRITER_MAX_ATTEMPTS", 10)

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
		MaxAttempts:  maxAttempts,
	}
	writer := &kafkaGoWriterImpl{kafka.NewWriter(config)}
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
		Timeout:       10 * time.Second,
		DualStack:     true,
		TLS:           tlsConfig,
		SASLMechanism: getSaslMechanism(),
	}

	return dialer
}

func getSaslMechanism() sasl.Mechanism {
	kafkaUsername := utils.Getenv("KAFKA_USERNAME", "")
	if kafkaUsername == "" {
		return nil
	}
	kafkaPassword := utils.GetenvOrFail("KAFKA_PASSWORD")
	saslType := utils.Getenv("KAFKA_SASL_TYPE", "scram")
	switch saslType {
	case "scram":
		mechanism, err := kafkaScram.Mechanism(kafkaScram.SHA512, kafkaUsername, kafkaPassword)
		if err != nil {
			panic(err)
		}
		return mechanism
	case "plain":
		mechanism := kafkaPlain.Mechanism{Username: kafkaUsername, Password: kafkaPassword}
		return mechanism
	}
	panic(fmt.Sprintf("Unknown sasl type '%s', options: {scram, plain}", saslType))
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
