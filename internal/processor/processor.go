package processor

import (
	"context"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/segmentio/kafka-go"
	"github.com/urbanindo/go-kafka-http-sink/config"
)

type httpHeader struct {
	key   string
	value string
}

type httpProcessor struct {
	http    *resty.Client
	url     string
	headers []httpHeader
}

func NewProcessor(conf *config.Config) httpProcessor {
	r := resty.New()
	headers := []httpHeader{}

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
		http:    r,
		url:     conf.HttpApiUrl,
		headers: headers,
	}
}

func (h *httpProcessor) Process(ctx context.Context, msg kafka.Message) error {
	r := h.http.NewRequest().SetContext(ctx)

	for _, header := range h.headers {
		r.SetHeader(header.key, header.value)
	}
	r.SetHeader("kafka_key", string(msg.Key))

	_, err := r.SetBody(msg.Value).
		Post(h.url)
	return err
}
