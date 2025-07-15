#!/bin/bash
./crdb-ory-load-test \
  --workload-config=local.yaml \
  --log-file=run.log \
  --tolerate-errors=true \
  --scope=hydra
