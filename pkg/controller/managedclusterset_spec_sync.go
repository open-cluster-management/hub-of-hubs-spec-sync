// Copyright (c) 2020 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package controller

import (
	"fmt"
	"k8s.io/apimachinery/pkg/api/equality"

	"github.com/jackc/pgx/v4/pgxpool"
	clusterv1beta1 "open-cluster-management.io/api/cluster/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func addManagedClusterSetController(mgr ctrl.Manager, databaseConnectionPool *pgxpool.Pool) error {
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&clusterv1beta1.ManagedClusterSet{}).
		Complete(&genericSpecToDBReconciler{
			client:                 mgr.GetClient(),
			databaseConnectionPool: databaseConnectionPool,
			log:                    ctrl.Log.WithName("managedclustersets-spec-syncer"),
			tableName:              "managedclustersets",
			finalizerName:          "hub-of-hubs.open-cluster-management.io/managedclustersets-cleanup",
			createInstance:         func() client.Object { return &clusterv1beta1.ManagedClusterSet{} },
			cleanStatus:            cleanManagedClusterSetStatus,
			areEqual:               areManagedClusterSetsEqual,
		}); err != nil {
		return fmt.Errorf("failed to add managed cluster set controller to the manager: %w", err)
	}

	return nil
}

func cleanManagedClusterSetStatus(instance client.Object) {
	managedClusterSet, ok := instance.(*clusterv1beta1.ManagedClusterSet)

	if !ok {
		panic("wrong instance passed to cleanManagedClusterSetStatus: not a ManagedClusterSet")
	}

	managedClusterSet.Status = clusterv1beta1.ManagedClusterSetStatus{}
}

func areManagedClusterSetsEqual(instance1, instance2 client.Object) bool {
	managedClusterSet1, ok1 := instance1.(*clusterv1beta1.ManagedClusterSet)
	managedClusterSet2, ok2 := instance2.(*clusterv1beta1.ManagedClusterSet)

	specMatch := ok1 && ok2 && equality.Semantic.DeepEqual(managedClusterSet1.Spec,
		managedClusterSet2.Spec)

	nameMatch := managedClusterSet1.ObjectMeta.Name == managedClusterSet2.ObjectMeta.Name

	return ok1 && ok2 && specMatch && nameMatch
}
