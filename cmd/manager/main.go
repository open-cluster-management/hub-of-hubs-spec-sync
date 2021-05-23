// Copyright (c) 2020 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package main

import (
	"context"
	"flag"
	"fmt"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)

	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/jackc/pgx/v4/pgxpool"

	"github.com/open-cluster-management/hub-of-hubs-spec-syncer/cmd/manager/tool"
	"github.com/open-cluster-management/hub-of-hubs-spec-syncer/pkg/apis"
	"github.com/open-cluster-management/hub-of-hubs-spec-syncer/pkg/controller"
	"github.com/open-cluster-management/hub-of-hubs-spec-syncer/version"

	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/operator-framework/operator-sdk/pkg/leader"
	"github.com/operator-framework/operator-sdk/pkg/log/zap"
	sdkVersion "github.com/operator-framework/operator-sdk/version"
	"github.com/spf13/pflag"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

// Change below variables to serve metrics on different host or port.
var (
	metricsHost               = "0.0.0.0"
	metricsPort         int32 = 8384
	operatorMetricsPort int32 = 8687
)
var log = logf.Log.WithName("cmd")

const (
	environmentVariableDatabaseUser     = "DB_USER"
	environmentVariableDatabasePassword = "DB_PASSWORD"
	environmentVariableDatabaseHost     = "DB_HOST"
	environmentVariableDatabasePort     = "DB_PORT"
	environmentVariableDatabaseName     = "DB_NAME"
)

func printVersion() {
	log.Info(fmt.Sprintf("Operator Version: %s", version.Version))
	log.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
	log.Info(fmt.Sprintf("Version of operator-sdk: %v", sdkVersion.Version))
}

func main() {
	// custom flags for the controler
	tool.ProcessFlags()
	// Add the zap logger flag set to the CLI. The flag set must
	// be added before calling pflag.Parse().
	pflag.CommandLine.AddFlagSet(zap.FlagSet())

	// Add flags registered by imported packages (e.g. glog and
	// controller-runtime)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)

	pflag.Parse()

	// Use a zap logr.Logger implementation. If none of the zap
	// flags are configured (or if the zap flag set is not being
	// used), this defaults to a production zap logger.
	//
	// The logger instantiated here can be changed to any logger
	// implementing the logr.Logger interface. This logger will
	// be propagated through the whole operator, generating
	// uniform and structured logs.
	logf.SetLogger(zap.Logger())

	printVersion()

	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		log.Error(err, "Failed to get watch namespace")
		os.Exit(1)
	}

	// Get database user
	if tool.Options.DatabaseUser == "" {
		found := false
		tool.Options.DatabaseUser, found = os.LookupEnv(environmentVariableDatabaseUser)
		if found {
			log.Info("Found:", "environment variable", environmentVariableDatabaseUser)
		}
	}

	// Get database password
	if tool.Options.DatabasePassword == "" {
		found := false
		tool.Options.DatabasePassword, found = os.LookupEnv(environmentVariableDatabasePassword)
		if found {
			log.Info("Found:", "environment variable", environmentVariableDatabasePassword)
		}
	}

	// Get database host
	if tool.Options.DatabaseHost == "" {
		found := false
		tool.Options.DatabaseHost, found = os.LookupEnv(environmentVariableDatabaseHost)
		if found {
			log.Info("Found:", "environment variable", environmentVariableDatabaseHost)
		}
	}

	// Get database port
	if tool.Options.DatabasePort == 0 {
		found := false
		databasePortAsString, found := os.LookupEnv(environmentVariableDatabasePort)
		if found {
			tool.Options.DatabasePort, err = strconv.Atoi(databasePortAsString)
			if err == nil {
				log.Info("Found:", "environment variable", environmentVariableDatabasePort)
			}
		}
	}

	// Get database name
	if tool.Options.DatabaseName == "" {
		found := false
		tool.Options.DatabaseName, found = os.LookupEnv(environmentVariableDatabaseName)
		if found {
			log.Info("Found:", "environment variable", environmentVariableDatabaseName)
		}
	}

	postgresURL := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=verify-full",
		tool.Options.DatabaseUser, url.QueryEscape(tool.Options.DatabasePassword), tool.Options.DatabaseHost,
		tool.Options.DatabasePort, tool.Options.DatabaseName)

	// open database
	dbConnectionPool, err := pgxpool.Connect(context.Background(), postgresURL)
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	defer dbConnectionPool.Close()

	var greeting string
	err = dbConnectionPool.QueryRow(context.Background(), "select 'Hello, world!'").Scan(&greeting)
	if err != nil {
		log.Error(err, "QueryRow failed")
		os.Exit(1)
	}

	log.Info(greeting)

	hubCfg, err := clientcmd.BuildConfigFromFlags("", tool.Options.HubConfigFilePathName)
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	ctx := context.TODO()
	// Become the leader before proceeding
	err = leader.Become(ctx, "hub-of-hubs-spec-syncer-lock")
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	// Set default manager options
	options := manager.Options{
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

	// Create a new manager to provide shared dependencies and start components
	mgr, err := manager.New(hubCfg, options)
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	log.Info("Registering Components.")

	// Setup Scheme for all resources
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	// Setup all Controllers
	if err := controller.AddToManager(mgr, tool.Options.HubConfigFilePathName); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	log.Info("Starting the Cmd.")

	// Start the Cmd
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Error(err, "Manager exited non-zero")
		os.Exit(1)
	}
}
