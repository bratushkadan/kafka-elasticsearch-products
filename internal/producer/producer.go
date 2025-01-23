package producer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"kafka/svcs/config"
	"kafka/svcs/pkg"
	"log"
	"math/rand/v2"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/twmb/franz-go/pkg/kgo"
)

var sellers = []string{"Danila", "Andrey", "Larisa"}

func getRandProduct() pkg.Product {
	name := gofakeit.ProductName()
	cat := gofakeit.ProductCategory()
	desc := gofakeit.ProductDescription()

	seller := sellers[int(rand.Int32())%len(sellers)]

	return pkg.Product{
		Id:          pkg.GenId(),
		Name:        name,
		Description: desc,
		Category:    cat,
		Seller:      seller,
	}
}

func randTickerMs(start, end int32) (<-chan struct{}, func()) {
	ch := make(chan struct{})
	done := make(chan struct{})

	go func() {
		defer close(ch)
		for {
			select {
			case <-done:
				return
			default:
				r := start + rand.Int32N(end-start)
				time.Sleep(time.Duration(r) * time.Millisecond)
				select {
				case <-done:
					return
				case ch <- struct{}{}:
				}
			}
		}
	}()

	return ch, func() {
		close(done)
	}
}

func NewKafkaClient() (*kgo.Client, error) {
	appConf := config.NewAppConf()
	// One client can both produce and consume!
	// Consuming can either be direct (no consumer group), or through a group. Below, we use a group.
	cl, err := kgo.NewClient(
		kgo.SeedBrokers(appConf.ClusterUrls()...),
		// kgo.ConsumerGroup("my-group-identifier"),
		// kgo.ConsumeTopics("foo"),
	)
	if err != nil {
		return nil, err
	}

	return cl, nil
}

func Run() {
	ch, cancel := randTickerMs(2000, 5000)
	defer cancel()

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

	for _ = range ch {
		var buf bytes.Buffer
		product := getRandProduct()
		_ = json.NewEncoder(&buf).Encode(product)
		record := &kgo.Record{Topic: config.NewAppConf().TopicName(), Value: buf.Bytes()}
		// This is **Asynchronous** produce! For synchronous produce use cl.ProduceSync.
		log.Print("trying to produce...")
		innerCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		cl.Produce(innerCtx, record, func(_ *kgo.Record, err error) {
			defer cancel()
			if err != nil {
				fmt.Printf("record had a produce error: %v\n", err)
			} else {
				fmt.Printf("produced a Product record: %+v\n", product)
			}
		})
	}
}
