package processor

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/linkedin/goavro/v2"
	"github.com/pkg/errors"
	"github.com/riferrei/srclient"
	"github.com/segmentio/kafka-go"
	"github.com/urbanindo/go-kafka-http-sink/config"
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
	errorWriter   ErrorWriter
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
			heads := strings.Split(head, ":")
			if len(heads) < 2 {
				panic("header should have key and value")
			}
			headers = append(headers, httpHeader{
				key:   heads[0],
				value: strings.Join(heads[1:], ""),
			})
		}
	}

	return httpProcessor{
		http:          r,
		url:           conf.HttpApiUrl,
		headers:       headers,
		logr:          logr,
		sr:            schemaRegistryClient,
		errorWriter:   NewErrorWriter(errorWriter),
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
		return err
	}

	if res.StatusCode() >= 300 && h.errorWriter != nil {
		if err := h.errorWriter.WriteError(ctx, msg.Key, &ErrorPayload{
			ResponseBody:    fmt.Sprintf("%s", res.Body()),
			ResponseCode:    res.StatusCode(),
			RequestBodyJSON: value,
		}); err != nil {
			return fmt.Errorf("error when writing to error topic: %v", err)
		}
		return fmt.Errorf("error from http with status code '%d': %s", res.StatusCode(), string(res.Body()))
	}

	h.logr.Debug("got " + res.Status() + " with body " + string(res.Body()))
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

	codec, err := goavro.NewCodecForStandardJSONFull(schema.Schema())
	if err != nil {
		return nil, fmt.Errorf("error initiate new avro codec: %s", err.Error())
	}
	native, _, err := codec.NativeFromBinary(msg.Value[5:])
	if err != nil {
		return nil, fmt.Errorf("error encode native from binary: %s", err.Error())
	}
	jsonStr, err := codec.TextualFromNative(nil, native)
	if err != nil {
		return nil, fmt.Errorf("error encode textual from native: %s", err.Error())
	}

	return jsonStr, nil
}

type ErrorPayload struct {
	ResponseBody    string          `json:"response_body"`
	ResponseCode    int             `json:"response_code"`
	RequestBodyJSON json.RawMessage `json:"request_body_json"`
}

type ErrorWriter interface {
	WriteError(ctx context.Context, key []byte, errPayload *ErrorPayload) error
}

func NewErrorWriter(kafkaWriter *kafka.Writer) ErrorWriter {
	return &errorWriter{
		writer: kafkaWriter,
	}
}

type errorWriter struct {
	writer *kafka.Writer
}

func (e *errorWriter) WriteError(ctx context.Context, key []byte, errPayload *ErrorPayload) error {
	value, err := json.Marshal(errPayload)
	if err != nil {
		return errors.Wrap(err, "WriteError: failed to marshal payload")
	}

	if err := e.writer.WriteMessages(ctx, kafka.Message{
		Key:   key,
		Value: value,
	}); err != nil {
		return errors.Wrap(err, "WriteError: failed write to kafka")
	}

	return nil
}
