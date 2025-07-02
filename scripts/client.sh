#!/bin/bash

docker run --rm -it \
  -e HYDRA_ADMIN_URL=http://hydra:4445 \
  --network hydraguide \
  oryd/hydra:v1.10.6 \
  clients create  \
    --id facebook-photo-backup \
    --secret some-secret \
    --grant-types authorization_code,refresh_token,client_credentials,implicit \
    --response-types token,code,id_token \
    --scope openid,offline,photos.read \
    --callbacks http://127.0.0.1:9010/callback
