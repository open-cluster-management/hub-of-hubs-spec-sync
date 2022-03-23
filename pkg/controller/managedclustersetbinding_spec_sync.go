// Copyright (c) 2020 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package controller

import (
	"fmt"

	"github.com/jackc/pgx/v4/pgxpool"
	clusterv1alpha1 "github.com/open-cluster-management/api/cluster/v1alpha1"
	"k8s.io/apimachinery/pkg/api/equality"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func addManagedClusterSetBindingController(mgr ctrl.Manager, databaseConnectionPool *pgxpool.Pool) error {
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&clusterv1alpha1.ManagedClusterSetBinding{}).
		Complete(&genericSpecToDBReconciler{
			client:                 mgr.GetClient(),
			databaseConnectionPool: databaseConnectionPool,
			log:                    ctrl.Log.WithName("managedclustersetbindings-spec-syncer"),
			tableName:              "managedclustersetbindings",
			finalizerName:          "hub-of-hubs.open-cluster-management.io/managedclustersetbindings-cleanup",
			createInstance:         func() client.Object { return &clusterv1alpha1.ManagedClusterSetBinding{} },
			cleanStatus:            cleanManagedClusterSetBindingsStatus,
			areEqual:               areManagedClusterSetBindingsEqual,
		}); err != nil {
		return fmt.Errorf("failed to add managed cluster set binding controller to the manager: %w", err)
	}

	return nil
}

func cleanManagedClusterSetBindingsStatus(instance client.Object) {
	_, ok := instance.(*clusterv1alpha1.ManagedClusterSetBinding)
	// ManagedClusterSetBinding has no status
	if !ok {
		panic("wrong instance passed to cleanManagedClusterSetBindingsStatus: not a ManagedClusterSetBinding")
	}
}

func areManagedClusterSetBindingsEqual(instance1, instance2 client.Object) bool {
	annotationMatch := equality.Semantic.DeepEqual(instance1.GetAnnotations(), instance2.GetAnnotations())

	managedClusterSetBinding1, ok1 := instance1.(*clusterv1alpha1.ManagedClusterSetBinding)
	managedClusterSetBinding2, ok2 := instance2.(*clusterv1alpha1.ManagedClusterSetBinding)

	specMatch := ok1 && ok2 && equality.Semantic.DeepEqual(managedClusterSetBinding1.Spec,
		managedClusterSetBinding2.Spec)

	return annotationMatch && specMatch
}
