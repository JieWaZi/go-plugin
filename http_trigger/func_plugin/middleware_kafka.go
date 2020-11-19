package func_plugin

import (
	"context"
	"github.com/SkyAPM/go2sky"
	language_agent "github.com/SkyAPM/go2sky/reporter/grpc/language-agent"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"time"
)

type Kafka struct {
	KafkaBrokers   []string
	KafkaDataTopic string
	DataPubClient  PubClient
	tracer         *go2sky.Tracer
	ctx            context.Context
}

func (k *Kafka) Init() error {
	k.DataPubClient = NewPubClient(kafka.WriterConfig{
		Brokers:  k.KafkaBrokers,
		Topic:    k.KafkaDataTopic,
		Balancer: &kafka.Hash{},
	})
	return nil
}

func (k *Kafka) SendDataMessage(key string, data []byte) error {
	span, _ := k.tracer.CreateExitSpan(k.ctx, "MQ Operation", k.KafkaBrokers[0], func(header string) error {
		return nil
	})
	span.Tag(go2sky.TagMQTopic, k.KafkaDataTopic)
	span.Tag(go2sky.TagMQBroker, k.KafkaBrokers[0])
	span.SetSpanLayer(language_agent.SpanLayer_MQ)
	err := k.DataPubClient.Send(context.Background(), kafka.Message{
		Key:   []byte(key),
		Value: data,
	})
	if err != nil {
		logrus.Error(err.Error())
		return err
	}
	defer func() {
		if err != nil {
			span.Error(time.Now(), "ERROR", err.Error())
		}
		span.End()
	}()

	return nil
}

func (k *Kafka) Close() error {
	err := k.DataPubClient.Close()
	if err != nil {
		logrus.Errorf("close data pubClient err:%s", err.Error())
		return err
	}

	return nil
}

func (k *Kafka) SetTrace(tracer *go2sky.Tracer, ctx context.Context) {
	k.ctx = ctx
	k.tracer = tracer
}

type PubClient interface {
	Send(ctx context.Context, msgs ...kafka.Message) error
	Close() error
}

type pubClient struct {
	writer *kafka.Writer
}

func NewPubClient(config kafka.WriterConfig) PubClient {
	writer := kafka.NewWriter(config)
	return &pubClient{writer: writer}
}

func (c *pubClient) Send(ctx context.Context, msgs ...kafka.Message) error {
	return c.writer.WriteMessages(ctx, msgs...)
}

func (c *pubClient) Close() error {
	return c.writer.Close()
}
