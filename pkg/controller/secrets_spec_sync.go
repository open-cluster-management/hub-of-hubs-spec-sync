// Copyright (c) 2020 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package controller

import (
	"fmt"
	"reflect"

	"github.com/jackc/pgx/v4/pgxpool"
	datatypes "github.com/open-cluster-management/hub-of-hubs-data-types"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

var (
	logger        = ctrl.Log.WithName("secret-spec-syncer")
	componentName = "secrets"

	secretPredicates = predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			// The object doesn't contain label "foo", so the event will be
			// ignored.
			if _, ok := e.MetaOld.GetAnnotations()[datatypes.HohSecretAnnotation]; !ok {
				return false
			}
			return true
		},

		CreateFunc: func(e event.CreateEvent) bool {
			if _, ok := e.Meta.GetAnnotations()[datatypes.HohSecretAnnotation]; !ok {
				return false
			}
			return true
		},

		DeleteFunc: func(e event.DeleteEvent) bool {
			if _, ok := e.Meta.GetAnnotations()[datatypes.HohSecretAnnotation]; !ok {
				return false
			}
			return true
		},
	}
)

func addSecretController(mgr ctrl.Manager, databaseConnectionPool *pgxpool.Pool) error {

	err := ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Secret{}).
		WithEventFilter(secretPredicates).
		Complete(&genericSpecToDBReconciler{
			client:                 mgr.GetClient(),
			databaseConnectionPool: databaseConnectionPool,
			log:                    logger,
			tableName:              componentName,
			finalizerName:          fmt.Sprintf("%s-cleanup", datatypes.HohSecretAnnotation),
			createInstance:         func() object { return &corev1.Secret{} },
			cleanStatus:            cleanSecretStatus,
			areEqual:               areSecretsEqual,
		})
	if err != nil {
		return fmt.Errorf("failed to add %sController to the manager: %w", componentName, err)
	}

	return nil
}

func cleanSecretStatus(instance object) {
	_, ok := instance.(*corev1.Secret)

	if !ok {
		panic("wrong instance passed to cleanConfigStatus: not corev1.Secret")
	}
}

func areSecretsEqual(instance1, instance2 object) bool {
	s1, ok1 := instance1.(*corev1.Secret)
	s2, ok2 := instance2.(*corev1.Secret)

	if !ok1 || !ok2 {
		return false
	}

	// only care about the Data and StringData field since the hive process only use these fields
	if !reflect.DeepEqual(s1.Data, s2.Data) {
		return false
	}

	if !reflect.DeepEqual(s1.StringData, s2.StringData) {
		return false
	}

	return true
}
