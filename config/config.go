package config

type AppConf struct {
	clusterUrls       []string
	topicName         string
	consumerGroupName string
}

func (c *AppConf) TopicName() string {
	return c.topicName
}
func (c *AppConf) ConsumerGroupName() string {
	return c.consumerGroupName
}

// Aka "seeds" (either in franz-go or in Kafka - I have no idea).
func (c *AppConf) ClusterUrls() []string {
	urls := make([]string, len(c.clusterUrls))
	copy(urls, c.clusterUrls)
	return urls
}

func NewAppConf() AppConf {
	return AppConf{
		clusterUrls:       []string{"localhost:9092"},
		topicName:         "foo",
		consumerGroupName: "go-foo-consumer-group",
	}
}
