package consumer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"kafka/svcs/config"
	"kafka/svcs/pkg"
	"log"
	"net/http"

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

		var products []pkg.Product

		fetches.EachPartition(func(p kgo.FetchTopicPartition) {
			// 2. Range iteration way to iterate over Records
			for _, record := range p.Records {
				var product pkg.Product
				if err := json.NewDecoder(bytes.NewReader(record.Value)).Decode(&product); err != nil {
					fmt.Printf("failed to unmarshal message from topic to a product: %v\n", err)
					continue
				}

				fmt.Printf("Read product message from a Kafka topic: %+v\n", product)
				products = append(products, product)
			}

			// 3. We can even use a second callback!

			/* p.EachRecord(func(record *kgo.Record) {
				fmt.Println(string(record.Value), "from a second callback!")
			}) */
		})

		resp, err := elasticSearchBulkInsert(context.Background(), products)
		if err != nil {
			log.Printf("failed to bulk insert data into elasticsearch: %v", err)
			log.Print(string(resp))
			continue
		}
		fmt.Println("bulk inserted to elasticsearch" + string(resp))

		fmt.Println("commit topic offsets")
		err = cl.CommitUncommittedOffsets(ctx)
		if err != nil {
			fmt.Printf("failed to commit offsets: %v\n", err)
		}
	}
}

func elasticSearchBulkInsert(ctx context.Context, products []pkg.Product) ([]byte, error) {
	var b bytes.Buffer

	for _, p := range products {
		b.WriteString(fmt.Sprintf(`{"index": {"_index": "floral-products", "_id": "%s"}}`, p.Id))
		b.WriteByte('\n')
		err := json.NewEncoder(&b).Encode(&struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Category    string `json:"category,omitempty"`
			Seller      string `json:"seller"`
		}{
			Name:        p.Name,
			Description: p.Description,
			Category:    p.Category,
			Seller:      p.Seller,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to encode product record: %w", err)
		}

		b.WriteByte('\n')
	}
	b.WriteByte('\n')

	req, err := http.NewRequestWithContext(ctx, "POST", "http://127.0.0.1:9200/_bulk", &b)
	if err != nil {
		return nil, fmt.Errorf("failed to create http request to perform bulk insert: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-ndjson")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode > 399 {
		return data, fmt.Errorf("request error")
	}

	return data, nil
}
