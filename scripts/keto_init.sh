#!/bin/bash

export DSN=cockroach://root@host.docker.internal:26257/keto?sslmode=disable
docker run -it --rm --network hydraguide  -e DSN=$DSN  -v ./keto:/home/ory oryd/keto  migrate up
