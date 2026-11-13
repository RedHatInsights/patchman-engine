package mqueue

import (
	"app/base/telemetry"
	"app/base/utils"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl"
	kafkaPlain "github.com/segmentio/kafka-go/sasl/plain"
	kafkaScram "github.com/segmentio/kafka-go/sasl/scram"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/trace"
)

type kafkaGoReaderImpl struct {
	kafka.Reader
}

func (t *kafkaGoReaderImpl) HandleMessages(ctx context.Context, n int, wait time.Duration, handler BatchHandler) {
	handleMessageBatches(ctx, t.Config().Topic, t.FetchMessage, t.CommitMessages, n, wait, handler)
}

func handleMessageBatches(ctx context.Context, topic string, fetch func(context.Context) (kafka.Message, error),
	commit func(context.Context, ...kafka.Message) error, n int, wait time.Duration, handler BatchHandler) {
	if n < 1 {
		n = 1
	}
	for {
		batch, err := readBatch(ctx, topic, fetch, n, wait)
		if err != nil {
			if readerStopped(err) {
				break
			}
			utils.LogError("err", err, "unable to read message batch from Kafka reader")
			panic(err)
		}

		handlerCtx := ctx
		if len(batch.app) == 1 {
			handlerCtx = batch.singleCtx
		}
		err = handler(handlerCtx, batch.app)
		batch.endSpans(err)
		if err != nil {
			utils.LogPanic("err", err, "Handler failed")
		}
		if err = commit(ctx, batch.raw...); err != nil {
			if readerStopped(err) {
				break
			}
			utils.LogError("err", err, "unable to commit kafka message batch")
			panic(err)
		}
	}
}

type fetchedBatch struct {
	app       []KafkaMessage
	raw       []kafka.Message
	spans     []trace.Span
	singleCtx context.Context
}

func (b *fetchedBatch) add(ctx context.Context, topic string, m kafka.Message) {
	if log.IsLevelEnabled(log.TraceLevel) {
		utils.LogTrace("count", len(m.Headers), "kafka message headers")
	}
	msgCtx, span := telemetry.ConsumerContext(ctx, topic, m.Headers)
	if len(b.app) == 0 {
		b.singleCtx = msgCtx
	}
	b.app = append(b.app, KafkaMessage{Key: m.Key, Value: m.Value, Headers: m.Headers})
	b.raw = append(b.raw, m)
	b.spans = append(b.spans, span)
}

func (b *fetchedBatch) endSpans(err error) {
	for _, span := range b.spans {
		telemetry.End(span, err)
	}
}

func readBatch(ctx context.Context, topic string, fetch func(context.Context) (kafka.Message, error),
	n int, wait time.Duration) (fetchedBatch, error) {
	batch := fetchedBatch{
		app:   make([]KafkaMessage, 0, n),
		raw:   make([]kafka.Message, 0, n),
		spans: make([]trace.Span, 0, n),
	}
	first, err := fetch(ctx)
	if err != nil {
		return fetchedBatch{}, err
	}
	batch.add(ctx, topic, first)
	if len(batch.app) >= n {
		return batch, nil
	}

	err = batch.fill(ctx, topic, fetch, n, wait)
	if err != nil {
		batch.endSpans(err)
		return fetchedBatch{}, err
	}
	return batch, nil
}

func (b *fetchedBatch) fill(ctx context.Context, topic string, fetch func(context.Context) (kafka.Message, error),
	n int, wait time.Duration) error {
	fetchCtx, cancel := context.WithTimeout(ctx, wait)
	defer cancel()
	for len(b.app) < n {
		m, err := fetch(fetchCtx)
		if err == nil {
			b.add(ctx, topic, m)
			continue
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return nil
		}
		return err
	}
	return nil
}

func readerStopped(err error) bool {
	return err.Error() == errContextCanceled || errors.Is(err, context.Canceled) || err == io.EOF
}

type kafkaGoWriterImpl struct {
	*kafka.Writer
}

func (t *kafkaGoWriterImpl) WriteMessages(ctx context.Context, msgs ...KafkaMessage) error {
	kafkaGoMessages := make([]kafka.Message, len(msgs))
	for i, m := range msgs {
		headers := telemetry.Inject(ctx, m.Headers)
		kafkaGoMessages[i] = kafka.Message{Key: m.Key, Value: m.Value, Headers: headers}
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
	return &kafkaGoWriterImpl{newKafkaWriterFromEnv(topic)}
}

func NewGzipKafkaWriterFromEnv(topic string, batchBytes int64) Writer {
	writer := newKafkaWriterFromEnv(topic)
	writer.BatchBytes = batchBytes
	writer.Compression = kafka.Gzip
	return &kafkaGoWriterImpl{writer}
}

func newKafkaWriterFromEnv(topic string) *kafka.Writer {
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
		MaxAttempts:  utils.CoreCfg.KafkaWriterMaxAttempts,
		// Messages can contain different number of systems for evalution. Use LeasBytes balancer
		// to balance partitions more equally. This way each partition should have _same_ number of systems
		// for evaluation.
		// NOTE: LeastBytes balancer won't work when we start add more producers in listener.
		// When using more producers, each producer have to create balanced messages.
		Balancer: &kafka.LeastBytes{},
	}
	return kafka.NewWriter(config)
}

// Init encrypting dialer if env var configured or return nil
func tryCreateSecuredDialerFromEnv() *kafka.Dialer {
	enableKafkaSsl := utils.CoreCfg.KafkaSslEnabled
	if !enableKafkaSsl {
		return nil
	}

	kafkaSslSkipVerify := utils.CoreCfg.KafkaSslSkipVerify
	// used only in development mode
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
	tlsConfig := tls.Config{RootCAs: caCertPool, MinVersion: tls.VersionTLS12}
	return &tlsConfig
}
