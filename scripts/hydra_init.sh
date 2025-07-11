#!/bin/bash

export DSN=cockroach://root@host.docker.internal:26257/hydra?sslmode=disable
docker run -it --rm --network hydraguide   oryd/hydra:v1.10.6  migrate sql --yes $DSN
