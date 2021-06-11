// Copyright (c) 2020 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package controller

import (
	"github.com/jackc/pgx/v4/pgxpool"
	appsv1 "github.com/open-cluster-management/governance-policy-propagator/pkg/apis/apps/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	ctrl "sigs.k8s.io/controller-runtime"
)

func AddPlacementRuleController(mgr ctrl.Manager, databaseConnectionPool *pgxpool.Pool) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.PlacementRule{}).
		Complete(&PlacementRuleReconciler{genericSpecToDBReconciler{
			client:                 mgr.GetClient(),
			databaseConnectionPool: databaseConnectionPool,
			log:                    ctrl.Log.WithName("placementrule-spec-syncer"),
			tableName:              "placementrules",
			finalizerName:          "hub-of-hubs.open-cluster-management.io/placementrule-cleanup",
			areEqual:               arePlacementRulesEqual,
		}})
}

type PlacementRuleReconciler struct {
	genericSpecToDBReconciler
}

func (r *PlacementRuleReconciler) Reconcile(request ctrl.Request) (ctrl.Result, error) {
	return r.reconcile(request, &appsv1.PlacementRule{}, &appsv1.PlacementRule{})
}

func arePlacementRulesEqual(instance1, instance2 object) bool {
	annotationMatch := equality.Semantic.DeepEqual(instance1.GetAnnotations(), instance2.GetAnnotations())

	placementRule1 := instance1.(*appsv1.PlacementRule)
	placementRule2 := instance2.(*appsv1.PlacementRule)
	specMatch := equality.Semantic.DeepEqual(placementRule1.Spec, placementRule2.Spec)

	return annotationMatch && specMatch
}
