// Copyright (c) 2020 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package tool

import (
	"github.com/spf13/pflag"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("cmd")

// HubOfHubsSpecSyncOptions for command line flag parsing
type HubOfHubsSpecSyncOptions struct {
	HubConfigFilePathName string
	DatabaseUserName      string
}

// Options default value
var Options = HubOfHubsSpecSyncOptions{}

// ProcessFlags parses command line parameters into Options
func ProcessFlags() {
	flag := pflag.CommandLine

	flag.StringVar(
		&Options.HubConfigFilePathName,
		"hub-cluster-configfile",
		Options.HubConfigFilePathName,
		"Configuration file pathname to hub kubernetes cluster",
	)

	flag.StringVar(
		&Options.DatabaseUserName,
		"database-user-name",
		Options.DatabaseUserName,
		"The user name to connect to the database",
	)
}
