package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"movieexample.com/rating/pkg/model"
)

const (
	fileName = "ratingsdata.json"
	topic    = "ratings"
)

func main() {
	fmt.Println("Creating a kafka producer")
	producer, err := kafka.NewProducer(&kafka.ConfigMap{"bootstrap.servers": "localhost"})
	if err != nil {
		panic(err)
	}
	defer producer.Close()

	fmt.Println("Reading rating events from file " + fileName)
	ratingEvents, err := readRatingEvents(fileName)
	if err != nil {
		panic(err)
	}

	if err := produceRatingEvents(topic, producer, ratingEvents); err != nil {
		panic(err)
	}

	timeout := 10 * time.Second
	fmt.Println("Waiting " + timeout.String() + " until all events get produced")

	producer.Flush(int(timeout.Milliseconds()))
}

func readRatingEvents(fileName string) ([]model.RatingEvent, error) {
	f, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var ratings []model.RatingEvent

	if err := json.NewDecoder(f).Decode(&ratings); err != nil {
		return nil, err
	}

	return ratings, err
}

func produceRatingEvents(topic string, producer *kafka.Producer, events []model.RatingEvent) error {
	for _, ratingEvents := range events {
		encodedEvent, err := json.Marshal(ratingEvents)
		if err != nil {
			return err
		}

		if err := producer.Produce(&kafka.Message{
			TopicPartition: kafka.TopicPartition{
				Topic:     &topic,
				Partition: kafka.PartitionAny,
			},
			Value: encodedEvent,
		}, nil); err != nil {
			return nil
		}
	}

	return nil
}
