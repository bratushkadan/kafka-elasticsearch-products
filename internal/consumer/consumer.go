package consumer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"kafka/svcs/config"
	"kafka/svcs/pkg"

	"github.com/twmb/franz-go/pkg/kgo"
)

func NewKafkaClient() (*kgo.Client, error) {
	appConf := config.NewAppConf()

	// One client can both produce and consume!
	// Consuming can either be direct (no consumer group), or through a group. Below, we use a group.
	cl, err := kgo.NewClient(
		kgo.SeedBrokers(appConf.ClusterUrls()...),
		kgo.ConsumerGroup(appConf.ConsumerGroupName()),
		kgo.ConsumeTopics(appConf.TopicName()),
	)
	if err != nil {
		return nil, err
	}

	return cl, nil

}

func Run() {
	cl, err := NewKafkaClient()
	if err != nil {
		panic(err)
	}
	defer cl.Close()

	ctx := context.Background()

	fmt.Println("Consumer starts polling...")
	for {
		fetches := cl.PollFetches(ctx)
		if errs := fetches.Errors(); len(errs) > 0 {
			panic(fmt.Sprint(errs))
		}

		// 1. First way to iterate over fetches
		/* iter := fetches.RecordIter()
		for !iter.Done() {
			record := iter.Next()
			fmt.Println(string(record.Value), "from an iterator!")
		} */

		fetches.EachPartition(func(p kgo.FetchTopicPartition) {
			// 2. Range iteration way to iterate over Records
			for _, record := range p.Records {
				var product pkg.Product
				if err := json.NewDecoder(bytes.NewReader(record.Value)).Decode(&product); err != nil {
					fmt.Printf("failed to unmarshal message from topic to a product: %v\n", err)
					continue
				}

				fmt.Printf("Read product message from a Kafka topic: %+v\n", product)
			}

			// 3. We can even use a second callback!

			/* p.EachRecord(func(record *kgo.Record) {
				fmt.Println(string(record.Value), "from a second callback!")
			}) */
		})

		err := cl.CommitUncommittedOffsets(ctx)
		if err != nil {
			fmt.Printf("failed to commit offsets: %v\n", err)
		}
	}
}
