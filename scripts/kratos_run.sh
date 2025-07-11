#!/bin/bash
export DSN=cockroach://root@host.docker.internal:26257/kratos?sslmode=disable

export SECRETS_SYSTEM="kratosSecretMustBeLong"
docker run -d \
  -v ./kratos:/etc/config/kratos \
  --network hydraguide \
  --name kratos \
  -p 4433:4433 \
  -p 4434:4434 \
  -e SECRETS_SYSTEM=$SECRETS_SYSTEM \
  -e DSN=$DSN \
  oryd/kratos serve -c /etc/config/kratos/kratos.yml
