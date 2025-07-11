#!/bin/bash
export DSN=cockroach://root@host.docker.internal:26257/hydra?sslmode=disable

export SECRETS_SYSTEM="hydraSecretMustBeLong"
docker run -d \
  --network hydraguide \
  --name hydra \
  -p 5444:4444 \
  -p 5445:4445 \
  -e SECRETS_SYSTEM=$SECRETS_SYSTEM \
  -e DSN=$DSN \
  -e URLS_SELF_ISSUER=http://localhost:5444/ \
  -e URLS_CONSENT=http://localhost:9020/consent \
  -e URLS_LOGIN=http://localhost:9020/login \
  oryd/hydra:v1.10.6 serve all --dangerous-force-http
