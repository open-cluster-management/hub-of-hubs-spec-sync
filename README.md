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

* DATABASE_URL
* WATCH_NAMESPACE

Set the `DATABASE_URL` according to the PostgreSQL URL format: `postgres://YourUserName:YourURLEscapedPassword@YourHostname:5432/YourDatabaseName?sslmode=verify-full`.

:exclamation-mark: Remember to URL-escape the password, you can do it in bash:

```
python -c "import sys, urllib as ul; print ul.quote_plus(sys.argv[1])" 'YourPassword'
```

```
./build/_output/bin/hub-of-hubs-spec-syncer --hub-cluster-configfile $TOP_HUB_CONFIG
```
