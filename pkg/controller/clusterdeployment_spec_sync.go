// Copyright (c) 2020 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package controller

import (
	"fmt"

	"github.com/jackc/pgx/v4/pgxpool"
	policiesv1 "github.com/open-cluster-management/governance-policy-propagator/pkg/apis/policy/v1"
	cdv1 "github.com/openshift/hive/apis/hive/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	controllerName = "clusterdeployment-spec-synce"
	componentName  = "clusterdeployments"
)

var (
	log = ctrl.Log.WithName(controllerName)
)

func addClusterDeploymentController(mgr ctrl.Manager, databaseConnectionPool *pgxpool.Pool) error {
	err := ctrl.NewControllerManagedBy(mgr).
		For(&policiesv1.Policy{}).
		Complete(&genericSpecToDBReconciler{
			client:                 mgr.GetClient(),
			databaseConnectionPool: databaseConnectionPool,
			log:                    ctrl.Log.WithName(controllerName),
			tableName:              componentName,
			finalizerName:          fmt.Sprintf("hub-of-hubs.open-cluster-management.io/%s-cleanup", componentName),
			createInstance:         func() object { return &cdv1.ClusterDeployment{} },
			cleanStatus:            cleanClusterDeploymentStatus,
			areEqual:               areClusterDeploymentEqual,
		})
	if err != nil {
		return fmt.Errorf("failed to add PolicyController to the manager: %w", err)
	}

	return nil
}

func cleanClusterDeploymentStatus(instance object) {
	ins, ok := instance.(*cdv1.ClusterDeployment)

	if !ok {
		panic(fmt.Sprintf("wrong instance passed to cleanConfigStatus: not hive/v1/%s", componentName))
	}

	ins.Status = cdv1.ClusterDeploymentStatus{}
}

func areClusterDeploymentEqual(instance1, instance2 object) bool {
	ins1, ok1 := instance1.(*cdv1.ClusterDeployment)
	ins2, ok2 := instance2.(*cdv1.ClusterDeployment)

	if !ok1 || !ok2 {
		return false
	}

	log.Info("need to add this compare func", ins1.GetName(), ins2.GetName())

	return true
}
