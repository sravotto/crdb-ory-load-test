#!/bin/bash
./crdb-ory-load-test \
  --duration-sec=10 \
  --workload-config=local.yaml \
  --log-file=run.log \
  --scope=kratos
