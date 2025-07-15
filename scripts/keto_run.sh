#!/bin/bash

export DSN=cockroach://root@host.docker.internal:26257/keto?sslmode=disable

docker run -d \
  --network hydraguide \
  --name keto \
  -v ./keto:/home/ory \
  -p 4466:4466 \
  -p 4467:4467 \
  -e DSN=$DSN \
  -e LOG_LEVEL=warn \
  oryd/keto 
