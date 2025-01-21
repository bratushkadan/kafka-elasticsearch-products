# Installation

## Prepare Kafka

```bash
docker run -d --name kafka-broker -p 9092:9092 apache/kafka:3.7.2
```

### Create Topics

```bash
docker exec --workdir /opt/kafka/bin -it kafka-broker ./kafka-topics.sh --bootstrap-server localhost:9092 --create --topic foo --partitions 12
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
