#!/bin/bash

/opt/kafka/bin/kafka-topics.sh --create --if-not-exists --bootstrap-server kafka:9092 --replication-factor 1 --partitions 1 --topic orders
/opt/kafka/bin/kafka-topics.sh --bootstrap-server kafka:9092 \
  --create --topic __consumer_offsets \
  --partitions 50 --replication-factor 1 \
  --config cleanup.policy=compact


sleep infinity