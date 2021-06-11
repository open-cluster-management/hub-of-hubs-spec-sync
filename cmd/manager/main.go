// Copyright (c) 2020 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/go-logr/logr"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/open-cluster-management/hub-of-hubs-spec-syncer/pkg/controller"
	"github.com/open-cluster-management/hub-of-hubs-spec-syncer/version"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/operator-framework/operator-sdk/pkg/leader"
	"github.com/operator-framework/operator-sdk/pkg/log/zap"
	sdkVersion "github.com/operator-framework/operator-sdk/version"
	"github.com/spf13/pflag"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
)

const (
	metricsHost                          = "0.0.0.0"
	metricsPort                    int32 = 8384
	environmentVariableDatabaseURL       = "DATABASE_URL"
)

func printVersion(log logr.Logger) {
	log.Info(fmt.Sprintf("Operator Version: %s", version.Version))
	log.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
	log.Info(fmt.Sprintf("Version of operator-sdk: %v", sdkVersion.Version))
}

// function to handle defers with exit, see https://stackoverflow.com/a/27629493/553720.
func doMain() int {
	pflag.CommandLine.AddFlagSet(zap.FlagSet())
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()

	ctrl.SetLogger(zap.Logger())
	log := ctrl.Log.WithName("cmd")

	printVersion(log)

	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		log.Error(err, "Failed to get watch namespace")
		return 1
	}

	// Get database URL
	databaseURL, found := os.LookupEnv(environmentVariableDatabaseURL)
	if found {
		log.Info("Found:", "environment variable", environmentVariableDatabaseURL)
	}

	// open database
	dbConnectionPool, err := pgxpool.Connect(context.Background(), databaseURL)
	if err != nil {
		log.Error(err, "")
		return 1
	}
	defer dbConnectionPool.Close()

	ctx := context.TODO()
	// Become the leader before proceeding
	err = leader.Become(ctx, "hub-of-hubs-spec-syncer-lock")
	if err != nil {
		log.Error(err, "")
		return 1
	}

	mgr, err := createManager(namespace, metricsHost, metricsPort, dbConnectionPool, log)
	if err != nil {
		return 1
	}

	log.Info("Starting the Cmd.")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		log.Error(err, "Manager exited non-zero")
		return 1
	}

	return 0
}

func createManager(namespace, metricsHost string, metricsPort int32, dbConnectionPool *pgxpool.Pool,
	log logr.Logger) (ctrl.Manager, error) {
	options := ctrl.Options{
		Namespace:          namespace,
		MetricsBindAddress: fmt.Sprintf("%s:%d", metricsHost, metricsPort),
	}

	// Add support for MultiNamespace set in WATCH_NAMESPACE (e.g ns1,ns2)
	// Note that this is not intended to be used for excluding namespaces, this is better done via a Predicate
	// Also note that you may face performance issues when using this with a high number of namespaces.
	// More Info: https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/cache#MultiNamespacedCacheBuilder
	if strings.Contains(namespace, ",") {
		options.Namespace = ""
		options.NewCache = cache.MultiNamespacedCacheBuilder(strings.Split(namespace, ","))
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), options)
	if err != nil {
		log.Error(err, "")
		return nil, err
	}

	log.Info("Registering Components.")
	if err := controller.AddToScheme(mgr.GetScheme()); err != nil {
		log.Error(err, "")
		return nil, err
	}

	if err := controller.AddControllers(mgr, dbConnectionPool); err != nil {
		log.Error(err, "")
		return nil, err
	}

	return mgr, nil
}

func main() {
	os.Exit(doMain())
}
