package controller

import (
	"fmt"

	"github.com/jackc/pgx/v4/pgxpool"
	"k8s.io/apimachinery/pkg/api/equality"
	subscriptionsv1 "open-cluster-management.io/multicloud-operators-subscription/pkg/apis/apps/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func addSubscriptionController(mgr ctrl.Manager, databaseConnectionPool *pgxpool.Pool) error {
	controlBuilder := ctrl.NewControllerManagedBy(mgr).For(&subscriptionsv1.Subscription{})
	controlBuilder = controlBuilder.WithEventFilter(generateNamespacePredicate())

	err := controlBuilder.Complete(&genericSpecToDBReconciler{
		client:                 mgr.GetClient(),
		databaseConnectionPool: databaseConnectionPool,
		log:                    ctrl.Log.WithName("subscriptions-spec-syncer"),
		tableName:              "subscriptions",
		finalizerName:          "hub-of-hubs.open-cluster-management.io/subscription-cleanup",
		createInstance:         func() client.Object { return &subscriptionsv1.Subscription{} },
		cleanStatus:            cleanSubscriptionStatus,
		areEqual:               areSubscriptionsEqual,
	})
	if err != nil {
		return fmt.Errorf("failed to add subscription controller to the manager: %w", err)
	}

	return nil
}

func areSubscriptionsEqual(instance1, instance2 client.Object) bool {
	// TODO: subscription come out as not equal because of package override field, check if it matters.
	annotationMatch := equality.Semantic.DeepEqual(instance1.GetAnnotations(), instance2.GetAnnotations())

	subscription1, ok1 := instance1.(*subscriptionsv1.Subscription)
	subscription2, ok2 := instance2.(*subscriptionsv1.Subscription)
	specMatch := ok1 && ok2 && equality.Semantic.DeepEqual(subscription1.Spec, subscription2.Spec)

	return annotationMatch && specMatch
}

func cleanSubscriptionStatus(instance client.Object) {
	subscription, ok := instance.(*subscriptionsv1.Subscription)
	if !ok {
		panic("wrong instance passed to cleanSubscriptionStatus: not a Subscription")
	}

	subscription.Status = subscriptionsv1.SubscriptionStatus{}
}
