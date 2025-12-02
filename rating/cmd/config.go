package main

type config struct {
	API              apiConfig              `yaml:"api"`
	ServiceDiscovery serviceDiscoveryConfig `yaml:"serviceDiscovery"`
	Kafka            kafkaConfig            `yaml:"kafka"`
}

type apiConfig struct {
	Port int `yaml:"port"`
}

type serviceDiscoveryConfig struct {
	Consul consulConfig `yaml:"consul"`
}
type consulConfig struct {
	Address string `yaml:"address"`
}

type kafkaConfig struct {
	Address string `yaml:"address"`
	Topic   string `yaml:"topic"`
	GroupID string `yaml:"groupID"`
}
