[comment]: # ( Copyright Contributors to the Open Cluster Management project )

# Hub-of-Hubs Spec Syncer
Red Hat Advanced Cluster Management Hub-of-Hubs Spec Syncer

(Syncer - the name taken from the [KCP project](https://github.com/kcp-dev/kcp/blob/main/contrib/demo/README.md#syncer))

## How it works

## Build to run locally

```
make build
```

## Run Locally

Set the following environment variables:

* DB_HOST
* DB_PORT
* DB_USER
* DB_NAME
* DB_PASSWORD
* WATCH_NAMESPACE

```
./build/_output/bin/hub-of-hubs-spec-syncer --hub-cluster-configfile $TOP_HUB_CONFIG
```
