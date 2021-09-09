package controller

import (
	"fmt"

	"github.com/jackc/pgx/v4/pgxpool"
	"k8s.io/apimachinery/pkg/api/equality"
	appsv1 "sigs.k8s.io/application/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func areApplicationsEqual(instance1, instance2 object) bool {
	annotationMatch := equality.Semantic.DeepEqual(instance1.GetAnnotations(), instance2.GetAnnotations())

	application1, ok1 := instance1.(*appsv1.Application)
	application2, ok2 := instance2.(*appsv1.Application)
	specMatch := ok1 && ok2 && equality.Semantic.DeepEqual(application1.Spec, application2.Spec)

	return annotationMatch && specMatch
}

func addApplicationController(mgr ctrl.Manager, databaseConnectionPool *pgxpool.Pool) error {
	err := ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.Application{}).
		Complete(&genericSpecToDBReconciler{
			client:                 mgr.GetClient(),
			databaseConnectionPool: databaseConnectionPool,
			log:                    ctrl.Log.WithName("application-spec-syncer"),
			tableName:              "applications",
			finalizerName:          "hub-of-hubs.open-cluster-management.io/application-cleanup",
			createInstance:         func() object { return &appsv1.Application{} },
			cleanStatus:            cleanApplicationStatus,
			areEqual:               areApplicationsEqual,
		})
	if err != nil {
		return fmt.Errorf("failed to add Application Controller to the manager: %w", err)
	}

	return nil
}

func cleanApplicationStatus(instance object) {
	application, ok := instance.(*appsv1.Application)
	if !ok {
		panic("wrong instance passed to cleanConfigStatus: not appsv1.Application")
	}

	application.Status = appsv1.ApplicationStatus{}
}
