#!/bin/bash

export DSN=cockroach://root@host.docker.internal:26257/kratos?sslmode=disable
docker run -it --rm --network hydraguide  -e DSN=$DSN  -v ./kratos:/home/ory oryd/kratos migrate sql "$DSN"
