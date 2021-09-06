package controller

import (
	"fmt"

	"github.com/jackc/pgx/v4/pgxpool"
	appsv1 "github.com/open-cluster-management/multicloud-operators-subscription/pkg/apis/apps/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	ctrl "sigs.k8s.io/controller-runtime"
)

func addSubscriptionController(mgr ctrl.Manager, databaseConnectionPool *pgxpool.Pool) error {
	err := ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.Subscription{}).
		Complete(&genericSpecToDBReconciler{
			client:                 mgr.GetClient(),
			databaseConnectionPool: databaseConnectionPool,
			log:                    ctrl.Log.WithName("subscriptions-spec-syncer"),
			tableName:              "subscriptions",
			finalizerName:          "hub-of-hubs.open-cluster-management.io/subscription-cleanup",
			createInstance:         func() object { return &appsv1.Subscription{} },
			cleanStatus:            cleanSubscriptionStatus,
			areEqual:               areSubscriptionsEqual,
		})
	if err != nil {
		return fmt.Errorf("failed to add Subscription Controller to the manager: %w", err)
	}

	return nil
}

func cleanSubscriptionStatus(instance object) {
	subscription, ok := instance.(*appsv1.Subscription)
	if !ok {
		panic("wrong instance passed to cleanConfigStatus: not appsv1.Subscription")
	}
	subscription.Status = appsv1.SubscriptionStatus{}
	return
}

func areSubscriptionsEqual(instance1, instance2 object) bool {
	annotationMatch := equality.Semantic.DeepEqual(instance1.GetAnnotations(), instance2.GetAnnotations())

	subscription1, ok1 := instance1.(*appsv1.Subscription)
	subscription2, ok2 := instance2.(*appsv1.Subscription)
	specMatch := ok1 && ok2 && equality.Semantic.DeepEqual(subscription1.Spec, subscription2.Spec)

	return annotationMatch && specMatch
}
