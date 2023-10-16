package mqueue

import (
	"app/base"
	"app/base/utils"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strings"
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
			utils.LogError("err", err.Error(), "unable to read message from Kafka reader")
			panic(err)
		}
		// At this level, all errors are fatal
		kafkaMessage := KafkaMessage{Key: m.Key, Value: m.Value}
		if err = handler(kafkaMessage); err != nil {
			utils.LogPanic("err", err.Error(), "Handler failed")
		}
		err = t.CommitMessages(base.Context, m)
		if err != nil {
			if err.Error() == errContextCanceled {
				break
			}
			utils.LogError("err", err.Error(), "unable to commit kafka message")
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
	kafkaAddress := utils.FailIfEmpty(utils.Cfg.KafkaAddress, "KAFKA_ADDRESS")
	kafkaGroup := utils.FailIfEmpty(utils.Cfg.KafkaGroup, "KAFKA_GROUP")
	minBytes := utils.Cfg.KafkaReaderMinBytes
	maxBytes := utils.Cfg.KafkaReaderMaxBytes
	maxAttempts := utils.Cfg.KafkaReaderMaxAttempts

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
	kafkaAddress := utils.FailIfEmpty(utils.Cfg.KafkaAddress, "KAFKA_ADDRESS")
	maxAttempts := utils.Cfg.KafkaWriterMaxAttempts

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
		// Messages can contain different number of systems for evalution. Use LeasBytes balancer
		// to balance partitions more equally. This way each partition should have _same_ number of systems
		// for evaluation.
		// NOTE: LeastBytes balancer won't work when we start add more producers in listener.
		// When using more producers, each producer have to create balanced messages.
		Balancer: &kafka.LeastBytes{},
	}
	writer := &kafkaGoWriterImpl{kafka.NewWriter(config)}
	return writer
}

// Init encrypting dialer if env var configured or return nil
func tryCreateSecuredDialerFromEnv() *kafka.Dialer {
	enableKafkaSsl := utils.Cfg.KafkaSslEnabled
	if !enableKafkaSsl {
		return nil
	}

	kafkaSslSkipVerify := utils.Cfg.KafkaSslSkipVerify
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
	if utils.Cfg.KafkaSaslType == nil {
		return nil
	}
	kafkaUsername := utils.Cfg.KafkaUsername
	if kafkaUsername == "" {
		return nil
	}
	kafkaPassword := utils.FailIfEmpty(utils.Cfg.KafkaPassword, "KAFKA_PASSWORD")
	saslType := strings.ToLower(*utils.Cfg.KafkaSaslType)
	switch saslType {
	case "scram", "scram-sha-512":
		mechanism, err := kafkaScram.Mechanism(kafkaScram.SHA512, kafkaUsername, kafkaPassword)
		if err != nil {
			panic(err)
		}
		return mechanism
	case "scram-sha-256":
		mechanism, err := kafkaScram.Mechanism(kafkaScram.SHA256, kafkaUsername, kafkaPassword)
		if err != nil {
			panic(err)
		}
		return mechanism
	case "plain":
		mechanism := kafkaPlain.Mechanism{Username: kafkaUsername, Password: kafkaPassword}
		return mechanism
	}
	panic(fmt.Sprintf("Unknown sasl type '%s', options: {scram, scram-sha-256, scram-sha-512, plain}", saslType))
}

func caCertTLSConfigFromEnv() *tls.Config {
	caCertPath := utils.FailIfEmpty(utils.Cfg.KafkaSslCert, "KAFKA_SSL_CERT")
	caCert, err := os.ReadFile(caCertPath)
	if err != nil {
		panic(err)
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	tlsConfig := tls.Config{RootCAs: caCertPool} // nolint:gosec
	return &tlsConfig
}
