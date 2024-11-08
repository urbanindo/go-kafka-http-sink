package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/segmentio/kafka-go"
	"github.com/urbanindo/go-kafka-http-sink/config"
	"github.com/urbanindo/go-kafka-http-sink/internal/processor"
	"github.com/urbanindo/go-kafka-http-sink/pkg/helper/logger"
	"go.uber.org/zap"
)

var (
	logr *zap.Logger
	code int
)

func main() {
	defer os.Exit(code)

	logr = logger.Named("go_kafka_http_sink")
	ctx, stop := signal.NotifyContext(
		context.Background(), syscall.SIGINT, syscall.SIGTERM,
	)
	defer stop()

	conf := config.Get()
	kafkaReader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{
			fmt.Sprintf("%s:%s", conf.KafkaConfig.Broker.Host, conf.KafkaConfig.Broker.Port),
		},
		Topic:   conf.KafkaConfig.Topic,
		GroupID: conf.KafkaConfig.ConsumerGroupName,
	})
	defer kafkaReader.Close()

	var (
		eWriter *kafka.Writer
		sWriter *kafka.Writer
	)

	if conf.KafkaConfig.SuccessTopic != nil {
		sWriter = kafka.NewWriter(kafka.WriterConfig{
			Brokers: []string{
				fmt.Sprintf("%s:%s", conf.KafkaConfig.Broker.Host, conf.KafkaConfig.Broker.Port),
			},
			Topic:    *conf.KafkaConfig.SuccessTopic,
			Balancer: kafka.Murmur2Balancer{},
		})
		logr.Debug("initiate kafka writer for success message")
	}

	if conf.KafkaConfig.ErrorTopic != nil {
		eWriter = kafka.NewWriter(kafka.WriterConfig{
			Brokers: []string{
				fmt.Sprintf("%s:%s", conf.KafkaConfig.Broker.Host, conf.KafkaConfig.Broker.Port),
			},
			Topic:    *conf.KafkaConfig.ErrorTopic,
			Balancer: kafka.Murmur2Balancer{},
		})
		logr.Debug("initiate kafka writer for error message")
	}

	proc := processor.NewProcessor(conf, logr, eWriter, sWriter)

	logr.Info("kafka http sink worker started. start for message...")
	for {
		msg, err := kafkaReader.ReadMessage(ctx)
		if err != nil {
			logr.Error(
				"failed to read message",
				zap.Any("message", msg),
				zap.Error(err),
			)
			continue
		}
		logr.Debug(
			"processing message",
			zap.Error(err),
			zap.String("payload", string(msg.Value)),
			zap.Int("offset", int(msg.Offset)),
		)

		if err := proc.Process(ctx, msg); err != nil {
			logr.Error(
				"failed to process message",
				zap.Any("message", msg),
				zap.Error(err),
			)
		}
	}
}
