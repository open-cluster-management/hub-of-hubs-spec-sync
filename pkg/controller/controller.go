// Copyright (c) 2020 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package controller

import (
	"github.com/jackc/pgx/v4/pgxpool"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// AddToManagerFuncs is a list of functions to add all Controllers to the Manager.
var AddToManagerFuncs []func(manager.Manager, *pgxpool.Pool) error

// AddToManager adds all Controllers to the Manager.
func AddToManager(m manager.Manager, databaseConnectionPool *pgxpool.Pool) error {
	for _, f := range AddToManagerFuncs {
		if err := f(m, databaseConnectionPool); err != nil {
			return err
		}
	}

	return nil
}
