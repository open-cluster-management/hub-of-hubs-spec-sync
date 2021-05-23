// Copyright (c) 2020 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package controller

import (
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"github.com/jackc/pgx/v4/pgxpool"
)

// AddToManagerFuncs is a list of functions to add all Controllers to the Manager
var AddToManagerFuncs []func(manager.Manager, *pgxpool.Pool) error

// AddToManager adds all Controllers to the Manager
func AddToManager(m manager.Manager, databaseConnectionPool *pgxpool.Pool) error {
	for _, f := range AddToManagerFuncs {
		if err := f(m, databaseConnectionPool); err != nil {
			return err
		}
	}
	return nil
}
