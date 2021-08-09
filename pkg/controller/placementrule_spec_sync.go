// Copyright (c) 2020 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package controller

import (
	"fmt"

	"github.com/jackc/pgx/v4/pgxpool"
	appsv1 "github.com/open-cluster-management/governance-policy-propagator/pkg/apis/apps/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	ctrl "sigs.k8s.io/controller-runtime"
)

func addPlacementRuleController(mgr ctrl.Manager, databaseConnectionPool *pgxpool.Pool) error {
	err := ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.PlacementRule{}).
		Complete(&genericSpecToDBReconciler{
			client:                 mgr.GetClient(),
			databaseConnectionPool: databaseConnectionPool,
			log:                    ctrl.Log.WithName("placementrule-spec-syncer"),
			tableName:              "placementrules",
			finalizerName:          "hub-of-hubs.open-cluster-management.io/placementrule-cleanup",
			createInstance:         func() object { return &appsv1.PlacementRule{} },
			cleanStatus:            cleanPlacementRuleStatus,
			areEqual:               arePlacementRulesEqual,
		})

	return fmt.Errorf("failed to add PlacementRule Controller to the manager: %w", err)
}

func cleanPlacementRuleStatus(instance object) {
	placementRule, ok := instance.(*appsv1.PlacementRule)

	if !ok {
		panic("wrong instance passed to cleanConfigStatus: not appsv1.PlacementRule")
	}

	placementRule.Status = appsv1.PlacementRuleStatus{}
}

func arePlacementRulesEqual(instance1, instance2 object) bool {
	annotationMatch := equality.Semantic.DeepEqual(instance1.GetAnnotations(), instance2.GetAnnotations())

	placementRule1, ok1 := instance1.(*appsv1.PlacementRule)
	placementRule2, ok2 := instance2.(*appsv1.PlacementRule)
	specMatch := ok1 && ok2 && equality.Semantic.DeepEqual(placementRule1.Spec, placementRule2.Spec)

	return annotationMatch && specMatch
}
