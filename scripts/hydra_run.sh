export DSN=cockroach://root@host.docker.internal:26257/hydra?sslmode=disable
#export SECRETS_SYSTEM=$(export LC_CTYPE=C; cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 32 | head -n 1)
#echo $SECRETS_SYSTEM > .secret
export SECRETS_SYSTEM=`cat .secret`
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
