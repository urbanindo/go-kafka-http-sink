package config

import (
	"fmt"
	"sync"

	"github.com/kelseyhightower/envconfig"
)

type Stage string

const (
	Local   Stage = "local"
	Dev           = "dev"
	Release       = "release"
	Prod          = "prod"
)

type KafkaBrokerConfig struct {
	Host string `envconfig:"HOST"`
	Port string `envconfig:"PORT"`
}

type KafkaConfig struct {
	Broker            KafkaBrokerConfig `envconfig:"BROKER"`
	Topic             string            `envconfig:"TOPIC"`
	ErrorTopic        *string           `envconfig:"ERROR_TOPIC"`
	SuccessTopic      *string           `envconfig:"SUCCESS_TOPIC"`
	ConsumerGroupName string            `envconfig:"CONSUMER_GROUP_NAME"`
}

type Config struct {
	KafkaConfig KafkaConfig `envconfig:"KAFKA"`
	HttpApiUrl  string      `envconfig:"HTTP_API_URL"`
	HttpHeaders *[]string   `envconfig:"HTTP_HEADERS"`
}

var cfgSync sync.Once
var confSingleton Config

// Get is Getter for Config
//
// Usage:
// ```
// import "github.com/urbanindo/regional-voucer/config"
//
// config.Get()
// ```
func Get() *Config {
	cfgSync.Do(func() {
		err := envconfig.Process("", &confSingleton)

		if err != nil {
			panic(fmt.Sprintln("Couldn't process config", err))
		}
	})

	return &confSingleton
}
