# Installation

## Prepare ElasticSearch

### Create `products` index

<details>
<summary>HTTP Request for creating `products` index</summary>

```bash
curl -X PUT http://127.0.0.1:9200/floral-products -H 'Content-Type: application/json' -d '
{
  "mappings": {
    "properties": {
      "name": {
        "type": "text",
        "analyzer": "autocomplete",
        "search_analyzer": "standard"
      },
      "description": {
        "type": "text",
        "analyzer": "standard"
      },
      "seller": {
        "type": "text",
        "analyzer": "autocomplete",
        "search_analyzer": "standard"
      },
      "category": {
        "type": "keyword"
      },
      "ad_priority": {
        "type": "integer"
      },
      "rating": {
        "type": "float"
      }
    }
  },
  "settings": {
    "analysis": {
      "analyzer": {
        "autocomplete": {
          "tokenizer": "autocomplete",
          "filter": [
            "lowercase"
          ]
        }
      },
      "tokenizer": {
        "autocomplete": {
          "type": "edge_ngram",
          "min_gram": 1,
          "max_gram": 20,
          "token_chars": [
            "letter",
            "digit"
          ]
        }
      }
    }
  }
}'
```

</details>

### Create `sellers` index

<details>
<summary>TODO:</summary>

</details>

### Adding products via REST API

<details>
<summary>Add product 1:</summary>

```bash
curl -X POST http://127.0.0.1:9200/floral-products/_doc/ -H 'Content-Type: application/json' -d'{
    "name": "Palm Tree",
    "description": "gorgeous palm tree",
    "seller": "danilas store for plants",
    "category": "tropical",
    "ad_priority": 0,
    "rating": 0
}'
```

</details>

<details>
<summary>Add product 2:</summary>


```bash
curl -X POST http://127.0.0.1:9200/floral-products/_doc/ -H 'Content-Type: application/json' -d'{
    "name": "ficus benjamina",
    "description": "Ficus tree has shown to have environmental benefits in urban areas like being a biomonitor. The plant also has allergens associated with it.",
    "seller": "danilas store for plants",
    "category": "tropical",
    "ad_priority": 0,
    "rating": 0
}'
```

</details>

### Updating products via REST API

<details>
<summary>Changing `ad_priority`:</summary>

```bash
curl -X POST http://localhost:9200/floral-products/_update/j8vch5QBeXeMBniXdVUR -H 'Content-Type: application/json' -d '{
  "doc": {
    "ad_priority": 10
  }
}'
```

</details>
<br>

Before updating `ad_score`, "Palm Tree" is the first search result with term "tree"; after updating `ad_score`, "ficus benjamina" is first search result (with its description containing search term "tree").

<details>
<summary>Query that demonstrates it:</summary>

```bash
curl -X POST http://localhost:9200/floral-products/_search -H 'Content-Type: application/json' -d '
{
  "query": {
    "function_score": {
      "query": {
        "multi_match": {
          "query": "tree",
          "fields": ["name", "description", "category", "seller"]
        }
      },
      "functions": [
        {
          "field_value_factor": {
            "field": "ad_priority",
            "factor": 0.6,
            "modifier": "log1p",
            "missing": 0
          }
        },
        {
          "field_value_factor": {
            "field": "rating",
            "factor": 0.4,
            "modifier": "sqrt",
            "missing": 0
          }
        }
      ],
      "boost_mode": "sum",
      "score_mode": "sum"
    }
  }
}
' | jq
```

</details>

### Querying

<details>
<summary>GET /products/_search</summary>

curl -X POST http://localhost:9200/floral-products/_search -H 'Content-Type: application/json' -d '
{
  "query": {
    "function_score": {
      "query": {
        "multi_match": {
          "query": "tree",
          "fields": ["name", "description", "category", "seller"]
        }
      },
      "functions": [
        {
          "field_value_factor": {
            "field": "ad_priority",
            "factor": 0.6,
            "modifier": "log1p",
            "missing": 0
          }
        },
        {
          "field_value_factor": {
            "field": "rating",
            "factor": 0.4,
            "modifier": "sqrt",
            "missing": 0
          }
        }
      ],
      "boost_mode": "sum",
      "score_mode": "sum"
    }
  }
}
' | jq

</details>

<details>
<summary><code>curl</code> example:</summary>

```bash
curl -X POST http://localhost:9200/floral-products/_search -H 'Content-Type: application/json' -d '
{
  "query": {
    "function_score": {
      "query": {
        "multi_match": {
          "query": "tropical",
          "fields": ["name", "description", "category", "seller"]
        }
      },
      "functions": [
        {
          "field_value_factor": {
            "field": "ad_priority",
            "factor": 0.6,
            "modifier": "log1p",
            "missing": 0
          }
        },
        {
          "field_value_factor": {
            "field": "rating",
            "factor": 0.4,
            "modifier": "sqrt",
            "missing": 0
          }
        }
      ],
      "boost_mode": "sum",
      "score_mode": "sum"
    }
  }
}
'
```
</details>

<details>
<summary>Querying request parameters explanation:</summary>

- **multi_match:**
  - Performs the initial full-text search based on terms entered by the user. It searches across specified fields, such as name and description.

- **function_score:**
  - Enhances the scoring of documents by incorporating additional criteria, like ad priority and rating.
  
- **field_value_factor:**
  - Adjusts document scores based on the value of numeric fields ad_priority and rating. The factor, modifier, and missing parameters can be tuned based on how much influence you want these fields to have.
  
- **boost_mode and score_mode:**
  - boost_mode determines how the field-based score (from field_value_factor) combines with the full-text search score. score_mode specifies how multiple boosting functions are combined.

</details>

## Prepare Kafka

```bash
docker run -d --name kafka-broker -p 9092:9092 apache/kafka:3.7.2
```

### Create Topics

```bash
docker exec --workdir /opt/kafka/bin -it kafka-broker ./kafka-topics.sh --bootstrap-server localhost:9092 --create --topic floral.products --partitions 2
```

### Consumer Groups

#### Create

A consumer group is automatically created when you start a Kafka consumer application with a specified group ID that doesn't already exist. There is no need to manually create a consumer group using a command.

#### List

```bash
docker exec --workdir /opt/kafka/bin -it kafka-broker ./kafka-consumer-groups.sh --bootstrap-server localhost:9092 --list
```

#### Describe

```bash
docker exec --workdir /opt/kafka/bin -it kafka-broker ./kafka-consumer-groups.sh --bootstrap-server localhost:9092 --describe --group <group_id>
```

#### Reset Offsets
   
```bash
docker exec --workdir /opt/kafka/bin -it kafka-broker ./kafka-consumer-groups.sh --bootstrap-server localhost:9092 --group <group_id> --reset-offsets --to-earliest --execute --topic <topic_name>
```
     
- The `--to-earliest` flag resets to the earliest offset; you can also reset to other points like `--to-latest`, `--to-current`, or specify offsets explicitly.

## Run App

There's no configured Partitioner for the Producer, so single producer will append messages to a single partition. This means that several consumers in this project's setup may only get a chance to consume simultaneously if 1) two producers were started and 2) two producers happened to write to partitions that are read by different consumers in a consumer group.

### Run Producer(s)

```bash
go run ./cmd/producer/main.go
```

### Run Consumer(s)

```bash
go run ./cmd/consumer/main.go
```
