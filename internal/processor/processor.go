package processor

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strings"
	"unicode"

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
	} else if isOtherDecoderbufsFormat(msg.Value) {
		// If Schema Registry is not available, check if the message is in decoderbufs format
		// Sanitize the payload (remove leading null bytes and re-serialize to clean JSON)
		value, err = sanitizePayload(msg.Value)
		if err != nil {
			return fmt.Errorf("payload sanitization failed: %v", err)
		}
	} else {
		value = msg.Value
	}

	r := h.http.NewRequest().SetContext(ctx)

	for _, header := range h.headers {
		r.SetHeader(header.key, header.value)
	}

	r.SetHeader("kafka_key", sanitizeKey(msg.Key))

	for _, msgHeader := range msg.Headers {
		// based on existing logic no need to add id header
		if msgHeader.Key != "id" {
			r.SetHeader(msgHeader.Key, string(msgHeader.Value))
		}
	}

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

func sanitizePayload(value []byte) ([]byte, error) {
	trimmedValue := bytes.TrimLeftFunc(value, func(r rune) bool {
		// Remove everything until we reach '{' or '[' that come from header of []byte to get exact body
		return !(r == '{' || r == '[')
	})

	var temp interface{}
	err := json.Unmarshal(trimmedValue, &temp)
	if err != nil {
		return nil, fmt.Errorf("invalid JSON payload: %w", err)
	}

	cleanedValue, _ := json.Marshal(temp)

	return cleanedValue, nil
}

// isOtherDecoderbufsFormat checks whether the given payload follows a specific decoderbufs-like format.
// The function assumes the payload contains a schema ID as the first 4 bytes.
// If the schema ID is 0, it considers the payload to match the decoderbufs format.
// This helps identify messages that require specific handling or sanitization.
func isOtherDecoderbufsFormat(value []byte) bool {
	if len(value) < 5 {
		return false
	}

	schemaID := binary.BigEndian.Uint32(value[:4])
	return schemaID == 0
}

// sanitizeKey ensures Kafka message keys are clean and valid for downstream use, such as in HTTP headers.
// Kafka keys may contain null bytes, non-printable, or invalid characters, causing issues in systems expecting clean strings.
// This function trims unnecessary characters and sanitizes the key only when needed, preserving compatibility and data integrity.
func sanitizeKey(key []byte) string {
	trimmed := strings.TrimSpace(string(bytes.Trim(key, "\x00")))
	var builder strings.Builder
	for _, r := range trimmed {
		if unicode.IsPrint(r) {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}
