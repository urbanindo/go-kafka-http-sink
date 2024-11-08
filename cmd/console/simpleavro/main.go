package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"

	"github.com/riferrei/srclient"
	"github.com/segmentio/kafka-go"
	"github.com/urbanindo/go-kafka-http-sink/config"
)

// {"type":"PL","user_id":2097886,"listing_id":"shs4411084","activation_date":"0001-01-01T00:00:00Z","activation_source":"QUOTA"}
type ComplexType struct {
	Type             string `json:"type"`
	UserId           int    `json:"user_id"`
	ListingID        string `json:"listing_id"`
	ActivationDate   string `json:"activation_date"`
	ActivationSource string `json:"activation_source,"`
}

func main() {

	conf := config.Get()

	// 1) Create the producer as you would normally do using Confluent's Go client
	p := kafka.NewWriter(kafka.WriterConfig{
		Brokers: []string{"kafka:9092"},
		Topic:   conf.KafkaConfig.Topic,
	})

	defer p.Close()

	// 2) Fetch the latest version of the schema, or create a new one if it is the first
	schemaRegistryClient := srclient.NewSchemaRegistryClient(*conf.KafkaConfig.SchemaRegistryUrl)
	schema, err := schemaRegistryClient.GetLatestSchema(conf.KafkaConfig.Topic + "-value")
	if schema == nil {
		panic(fmt.Sprintf("Error creating the schema %s", err))
	}
	schemaIDBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(schemaIDBytes, uint32(schema.ID()))
	fmt.Println(schema.SchemaType())

	// 3) Serialize the record using the schema provided by the client,
	// making sure to include the schema id as part of the record.
	newComplexType := ComplexType{Type: "PL", UserId: 2097886, ListingID: "shs4411084", ActivationDate: "0001-01-01T00:00:00Z", ActivationSource: "QUOTA"}
	value, _ := json.Marshal(newComplexType)
	fmt.Println(string(value))
	native, _, b := schema.Codec().NativeFromTextual([]byte(`{"payload":{"type":{"string":"PL"},"user_id":{"int":2097886}}}`))
	if b != nil {
		panic(b)
	}
	fmt.Println(native)
	valueBytes, _ := schema.Codec().BinaryFromNative(nil, native)

	var recordValue []byte
	recordValue = append(recordValue, byte(0))
	recordValue = append(recordValue, schemaIDBytes...)
	recordValue = append(recordValue, valueBytes...)

	key := "abcd1234"
	err = p.WriteMessages(context.TODO(), kafka.Message{
		Key:   []byte(key),
		Value: recordValue,
	})
	if err != nil {
		panic(err)
	}
}
