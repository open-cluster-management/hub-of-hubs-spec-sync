package controller

import (
	"fmt"

	"github.com/jackc/pgx/v4/pgxpool"
	appsv1 "github.com/open-cluster-management/multicloud-operators-channel/pkg/apis/apps/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	ctrl "sigs.k8s.io/controller-runtime"
)

func addChannelController(mgr ctrl.Manager, databaseConnectionPool *pgxpool.Pool) error {
	err := ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.Channel{}).
		Complete(&genericSpecToDBReconciler{
			client:                 mgr.GetClient(),
			databaseConnectionPool: databaseConnectionPool,
			log:                    ctrl.Log.WithName("channel-spec-syncer"),
			tableName:              "channels",
			finalizerName:          "hub-of-hubs.open-cluster-management.io/channel-cleanup",
			createInstance:         func() object { return &appsv1.Channel{} },
			cleanStatus:            cleanChannelStatus,
			areEqual:               areChannelsEqual,
		})
	if err != nil {
		return fmt.Errorf("failed to add Channel Controller to the manager: %w", err)
	}

	return nil
}

func cleanChannelStatus(instance object) {
	channel, ok := instance.(*appsv1.Channel)
	if !ok {
		panic("wrong instance passed to cleanConfigStatus: not appsv1.Channel")
	}
	channel.Status = appsv1.ChannelStatus{}
	return
}

func areChannelsEqual(instance1, instance2 object) bool {
	annotationMatch := equality.Semantic.DeepEqual(instance1.GetAnnotations(), instance2.GetAnnotations())

	channel1, ok1 := instance1.(*appsv1.Channel)
	channel2, ok2 := instance2.(*appsv1.Channel)
	specMatch := ok1 && ok2 && equality.Semantic.DeepEqual(channel1.Spec, channel2.Spec)

	return annotationMatch && specMatch
}
