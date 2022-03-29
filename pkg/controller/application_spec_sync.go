// Copyright (c) 2020 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package controller

import (
	"fmt"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stolostron/hub-of-hubs-spec-sync/pkg/helpers"
	"k8s.io/apimachinery/pkg/api/equality"
	appsv1beta1 "sigs.k8s.io/application/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

func addApplicationController(mgr ctrl.Manager, databaseConnectionPool *pgxpool.Pool) error {
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&appsv1beta1.Application{}).
		WithEventFilter(predicate.NewPredicateFuncs(func(object client.Object) bool {
			return !helpers.HasAnnotation(object, hubOfHubsLocalResource)
		})).
		Complete(&genericSpecToDBReconciler{
			client:                 mgr.GetClient(),
			databaseConnectionPool: databaseConnectionPool,
			log:                    ctrl.Log.WithName("applications-spec-syncer"),
			tableName:              "applications",
			finalizerName:          "hub-of-hubs.open-cluster-management.io/application-cleanup",
			createInstance:         func() client.Object { return &appsv1beta1.Application{} },
			cleanStatus:            cleanApplicationStatus,
			areEqual:               areApplicationsEqual,
		}); err != nil {
		return fmt.Errorf("failed to add application controller to the manager: %w", err)
	}

	return nil
}

func cleanApplicationStatus(instance client.Object) {
	application, ok := instance.(*appsv1beta1.Application)
	if !ok {
		panic("wrong instance passed to cleanApplicationStatus: not an Application")
	}

	application.Status = appsv1beta1.ApplicationStatus{}
}

func areApplicationsEqual(instance1, instance2 client.Object) bool {
	annotationMatch := equality.Semantic.DeepEqual(instance1.GetAnnotations(), instance2.GetAnnotations())

	application1, ok1 := instance1.(*appsv1beta1.Application)
	application2, ok2 := instance2.(*appsv1beta1.Application)
	specMatch := ok1 && ok2 && equality.Semantic.DeepEqual(application1.Spec, application2.Spec)

	return annotationMatch && specMatch
}
