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
	log "github.com/sirupsen/logrus"
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
		if log.IsLevelEnabled(log.TraceLevel) {
			utils.LogTrace("count", len(m.Headers), "kafka message headers")
			for _, h := range m.Headers {
				utils.LogTrace("key", h.Key, "value", string(h.Value), "kafka message header")
			}
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
	kafkaGroup := utils.FailIfEmpty(utils.CoreCfg.KafkaGroup, "KAFKA_GROUP")
	minBytes := utils.CoreCfg.KafkaReaderMinBytes
	maxBytes := utils.CoreCfg.KafkaReaderMaxBytes
	maxAttempts := utils.CoreCfg.KafkaReaderMaxAttempts

	config := kafka.ReaderConfig{
		Brokers:     utils.CoreCfg.KafkaServers,
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
	maxAttempts := utils.CoreCfg.KafkaWriterMaxAttempts

	config := kafka.WriterConfig{
		Brokers: utils.CoreCfg.KafkaServers,
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
	enableKafkaSsl := utils.CoreCfg.KafkaSslEnabled
	if !enableKafkaSsl {
		return nil
	}

	kafkaSslSkipVerify := utils.CoreCfg.KafkaSslSkipVerify
	tlsConfig := &tls.Config{InsecureSkipVerify: kafkaSslSkipVerify} // nolint:gosec
	if !kafkaSslSkipVerify {
		tlsConfig = caCertTLSConfig()
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
	if utils.CoreCfg.KafkaSaslType == nil {
		return nil
	}
	kafkaUsername := utils.CoreCfg.KafkaUsername
	if kafkaUsername == "" {
		return nil
	}
	kafkaPassword := utils.FailIfEmpty(utils.CoreCfg.KafkaPassword, "KAFKA_PASSWORD")
	saslType := strings.ToLower(*utils.CoreCfg.KafkaSaslType)
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

func caCertTLSConfig() *tls.Config {
	caCertPool, _ := x509.SystemCertPool()
	if caCertPool == nil {
		caCertPool = x509.NewCertPool()
	}
	if len(utils.CoreCfg.KafkaSslCert) > 0 {
		caCert, err := os.ReadFile(utils.CoreCfg.KafkaSslCert)
		if err != nil {
			panic(err)
		}
		caCertPool.AppendCertsFromPEM(caCert)
	}
	tlsConfig := tls.Config{RootCAs: caCertPool} // nolint:gosec
	return &tlsConfig
}
