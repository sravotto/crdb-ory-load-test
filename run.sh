#!/bin/bash
./crdb-ory-load-test \
  --duration-sec=60 \
  --read-ratio=200 \
  --workload-config=local.yaml \
  --log-file=run.log \
  --scope=hydra
