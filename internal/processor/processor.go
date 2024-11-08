package processor

import (
	"context"
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/riferrei/srclient"
	"github.com/segmentio/kafka-go"
	"github.com/urbanindo/go-kafka-http-sink/config"
	"github.com/urbanindo/go-kafka-http-sink/pkg/dto"
	"go.uber.org/zap"
)

type httpHeader struct {
	key   string
	value string
}

type httpProcessor struct {
	http          *resty.Client
	sr            *srclient.SchemaRegistryClient
	url           string
	logr          *zap.Logger
	headers       []httpHeader
	errorWriter   *kafka.Writer
	successWriter *kafka.Writer
}

func NewProcessor(conf *config.Config, logr *zap.Logger, errorWriter *kafka.Writer, successWriter *kafka.Writer) httpProcessor {
	r := resty.New()
	headers := []httpHeader{}
	var schemaRegistryClient *srclient.SchemaRegistryClient

	if conf.KafkaConfig.SchemaRegistryUrl != nil {
		schemaRegistryClient = srclient.NewSchemaRegistryClient(
			*conf.KafkaConfig.SchemaRegistryUrl,
		)
	}

	if conf.HttpHeaders != nil {
		for _, head := range *conf.HttpHeaders {
			heads := strings.Split(head, "=")
			if len(heads) != 2 {
				panic("header should have key and value")
			}
			headers = append(headers, httpHeader{
				key:   heads[0],
				value: heads[1],
			})
		}
	}

	return httpProcessor{
		http:          r,
		url:           conf.HttpApiUrl,
		headers:       headers,
		logr:          logr,
		sr:            schemaRegistryClient,
		errorWriter:   errorWriter,
		successWriter: successWriter,
	}
}

func (h *httpProcessor) Process(ctx context.Context, msg kafka.Message) error {
	var value []byte
	var err error
	if h.sr != nil {
		value, err = convertFromSchemaRegistry(h.sr, msg)

		if err != nil {
			return err
		}
	} else {
		value = msg.Value
	}

	r := h.http.NewRequest().SetContext(ctx)

	for _, header := range h.headers {
		r.SetHeader(header.key, header.value)
	}
	r.SetHeader("kafka_key", string(msg.Key))

	res, err := r.SetBody(value).
		Post(h.url)

	if err != nil {
		if h.errorWriter != nil {
			err2 := h.errorWriter.WriteMessages(ctx, kafka.Message{
				Key:   msg.Key,
				Value: []byte(fmt.Sprint("Failed with error", err)),
			})
			if err2 != nil {
				return fmt.Errorf("error when writing to error topic: %v", err2)
			}
		}
		return err
	}

	if res.StatusCode() >= 300 && h.errorWriter != nil {
		err = h.errorWriter.WriteMessages(ctx, kafka.Message{
			Key:   msg.Key,
			Value: []byte(fmt.Sprintf("Failed from http with status code '%d': %s", res.StatusCode(), string(res.Body()))),
		})
		if err != nil {
			return fmt.Errorf("error when writing to error topic: %v", err)
		}
		return fmt.Errorf("error from http with status code '%d': %s", res.StatusCode(), string(res.Body()))
	}

	if h.successWriter != nil {
		return h.successWriter.WriteMessages(ctx, kafka.Message{
			Key:   msg.Key,
			Value: value,
		})
	}
	return nil
}

func convertFromSchemaRegistry(sr *srclient.SchemaRegistryClient, msg kafka.Message) ([]byte, error) {
	schemaID := binary.BigEndian.Uint32(msg.Value[1:5])
	schema, err := sr.GetSchema(int(schemaID))

	if err != nil {
		return []byte{}, fmt.Errorf("error getting the schema with id '%d' %s", schemaID, err)
	}

	native, _, _ := schema.Codec().NativeFromBinary(msg.Value[5:])

	return dto.ConvertKafkaNativeToJson(native)
}
