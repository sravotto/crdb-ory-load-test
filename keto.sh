#!/bin/bash
./crdb-ory-load-test \
  --duration-sec=120 \
  --workload-config=local.yaml \
  --log-file=run.log \
  --scope=keto
