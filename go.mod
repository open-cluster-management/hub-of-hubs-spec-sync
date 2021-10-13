module github.com/open-cluster-management/hub-of-hubs-spec-sync

go 1.16

require (
	github.com/go-logr/logr v0.2.1
	github.com/go-logr/zapr v0.2.0 // indirect
	github.com/google/go-cmp v0.5.4 // indirect
	github.com/jackc/pgx/v4 v4.11.0
	github.com/onsi/gomega v1.10.2 // indirect
	github.com/open-cluster-management/governance-policy-propagator v0.0.0-20210520203318-a78632de1e26
	github.com/open-cluster-management/hub-of-hubs-data-types v0.1.0
	github.com/open-cluster-management/hub-of-hubs-data-types/apis/config v0.1.0
	golang.org/x/tools v0.1.5 // indirect
	k8s.io/api v0.20.5
	k8s.io/apimachinery v0.20.5
	k8s.io/client-go v12.0.0+incompatible
	sigs.k8s.io/controller-runtime v0.6.2
)

replace (
	github.com/open-cluster-management/hub-of-hubs-data-types => /Users/ianzhang/golang/src/hoh/src/hub-of-hubs-data-types
	k8s.io/client-go => k8s.io/client-go v0.20.5
)
