// Copyright (c) 2020 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package sync

import (
	policiesv1 "github.com/open-cluster-management/governance-policy-propagator/pkg/apis/policy/v1"

	"github.com/jackc/pgx/v4/pgxpool"
	"k8s.io/apimachinery/pkg/api/equality"
	ctrl "sigs.k8s.io/controller-runtime"
)

func AddPlacementBindingController(mgr ctrl.Manager, databaseConnectionPool *pgxpool.Pool) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&policiesv1.PlacementBinding{}).
		Complete(&PlacementBindingReconciler{genericSpecToDBReconciler{
			client:                 mgr.GetClient(),
			databaseConnectionPool: databaseConnectionPool,
			log:                    ctrl.Log.WithName("placementbinding-spec-syncer"),
			tableName:              "placementbindings",
			finalizerName:          "hub-of-hubs.open-cluster-management.io/placementbinding-cleanup",
			areEqual:               arePlacementBindingsEqual}})
}

type PlacementBindingReconciler struct {
	genericSpecToDBReconciler
}

func (r *PlacementBindingReconciler) Reconcile(request ctrl.Request) (ctrl.Result, error) {
	return r.reconcile(request, &policiesv1.PlacementBinding{}, &policiesv1.PlacementBinding{})
}

func arePlacementBindingsEqual(instance1, instance2 object) bool {
	annotationMatch := equality.Semantic.DeepEqual(instance1.GetAnnotations(), instance2.GetAnnotations())

	placementBinding1 := instance1.(*policiesv1.PlacementBinding)
	placementBinding2 := instance2.(*policiesv1.PlacementBinding)

	placementRefMatch := equality.Semantic.DeepEqual(placementBinding1.PlacementRef, placementBinding2.PlacementRef)
	subjectsMatch := equality.Semantic.DeepEqual(placementBinding1.Subjects, placementBinding2.Subjects)

	return annotationMatch && placementRefMatch && subjectsMatch
}
