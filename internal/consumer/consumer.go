package consumer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"kafka/svcs/config"
	"kafka/svcs/pkg"
	"log"
	"net/http"
	"time"

	"github.com/bratushkadan/back-goff/pkg/backoff"
	"github.com/twmb/franz-go/pkg/kgo"
)

func NewKafkaClient() (*kgo.Client, error) {
	appConf := config.NewAppConf()

	// One client can both produce and consume!
	// Consuming can either be direct (no consumer group), or through a group. Below, we use a group.
	cl, err := kgo.NewClient(
		kgo.SeedBrokers(appConf.BrokerUrls()...),
		kgo.ConsumerGroup(appConf.ConsumerGroupName()),
		kgo.ConsumeTopics(appConf.TopicName()),
		kgo.DisableAutoCommit(),
		kgo.SessionTimeout(3*time.Second),
		kgo.HeartbeatInterval(1500*time.Millisecond),
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

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := cl.Ping(ctx); err != nil {
		log.Fatal(fmt.Errorf("failed to ping brokers: %w", err))
	}
	ctx = context.Background()

	backoff := backoff.New(backoff.Conf{BaseStart: time.Second, BaseMax: 30 * time.Second, JitterMin: 200 * time.Millisecond, JitterMax: 1 * time.Second, Factor: 2.0})
	for {
		log.Print("fetching")

		// startOffsets := cl.UncommittedOffsets()
		_ = cl.UncommittedOffsets()
		// log.Printf("before poll fetches cl.MarkedOffsets() = %+v\n", cl.MarkedOffsets())
		fetches := cl.PollFetches(ctx)
		fmt.Printf("commited offsets: %+v\n", cl.CommittedOffsets())
		// log.Printf("after poll fetches cl.MarkedOffsets() = %+v\n", cl.MarkedOffsets())
		if fetches.IsClientClosed() {
			break
		}
		if errs := fetches.Errors(); len(errs) > 0 {
			panic(fmt.Sprint(errs))
		}

		var products []pkg.Product

		fetches.EachPartition(func(p kgo.FetchTopicPartition) {
			for _, record := range p.Records {
				log.Printf("topic %s partition %d offset %d \n", p.Topic, p.Partition, record.Offset)
				var product pkg.Product
				if err := json.NewDecoder(bytes.NewReader(record.Value)).Decode(&product); err != nil {
					fmt.Printf("failed to unmarshal message from topic to a product: %v\n", err)
					continue
				}

				fmt.Printf("Read product message from a Kafka topic: %+v\n", product)
				products = append(products, product)
			}
		})

		resp, err := elasticSearchBulkInsert(context.Background(), products)
		if err != nil {
			log.Printf("failed to bulk insert data into elasticsearch: %v", err)
			log.Print(string(resp))
			// Reset offsets to retry fetching data that couldn't be processed.
			cl.SetOffsets(cl.CommittedOffsets())

			b := backoff.GetIncr()
			log.Printf("Retrying poll after %.2fs", b.Seconds())
			time.Sleep(b)
			continue
		}
		backoff.Reset()
		fmt.Println("bulk inserted to elasticsearch" + string(resp))

		fmt.Println("commit topic offsets")
		err = cl.CommitUncommittedOffsets(ctx)
		if err != nil {
			fmt.Printf("failed to commit offsets: %v\n", err)
		}
	}
}

func elasticSearchBulkInsert(ctx context.Context, products []pkg.Product) ([]byte, error) {
	return nil, errors.New("fake error to make elastiSeachBulkInsert fail")

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
