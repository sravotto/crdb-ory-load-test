## Hydra local test

Assuming:

* CRDB running locally, insecure mode.

Create hydra DB.

Run migration script:

```text
./hydra_init.sh 
```

Run hydra server:

```text
./network.sh
./hydra_run.sh 
```

From the top dir (`..`), compile:

```text
make
```

Run test:

```text
./run.sh
```
