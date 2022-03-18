// Copyright (c) 2020 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package controller

import (
	"fmt"

	"github.com/jackc/pgx/v4/pgxpool"
	policiesv1 "github.com/open-cluster-management/governance-policy-propagator/api/v1"
	"github.com/open-cluster-management/governance-policy-propagator/controllers/common"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func addPolicyController(mgr ctrl.Manager, databaseConnectionPool *pgxpool.Pool) error {
	err := ctrl.NewControllerManagedBy(mgr).
		For(&policiesv1.Policy{}).
		Complete(&genericSpecToDBReconciler{
			client:                 mgr.GetClient(),
			databaseConnectionPool: databaseConnectionPool,
			log:                    ctrl.Log.WithName("policy-spec-syncer"),
			tableName:              "policies",
			finalizerName:          "hub-of-hubs.open-cluster-management.io/policy-cleanup",
			createInstance:         func() client.Object { return &policiesv1.Policy{} },
			processInstance:        processPolicyInstance,
			cleanStatus:            cleanPolicyStatus,
			areEqual:               arePoliciesEqual,
		})
	if err != nil {
		return fmt.Errorf("failed to add PolicyController to the manager: %w", err)
	}

	return nil
}

func cleanPolicyStatus(instance client.Object) {
	policy, ok := instance.(*policiesv1.Policy)

	if !ok {
		panic("wrong instance passed to cleanConfigStatus: not policiesv1.Policy")
	}

	policy.Status = policiesv1.PolicyStatus{}
}

func processPolicyInstance(instance client.Object) client.Object {
	policy, ok := instance.(*policiesv1.Policy)

	if !ok {
		panic("wrong instance passed to processPolicyInstance: not policiesv1.Policy")
	}

	annotations := policy.GetAnnotations()
	if annotations == nil {
		return instance
	}

	if _, ok := annotations[hubOfHubsLocalPolicy]; ok {
		return nil
	}

	return instance
}

func arePoliciesEqual(instance1, instance2 client.Object) bool {
	policy1, ok1 := instance1.(*policiesv1.Policy)
	policy2, ok2 := instance2.(*policiesv1.Policy)

	if !ok1 || !ok2 {
		return false
	}

	// TODO handle Template comparison later
	policy1WithoutTemplates := policy1.DeepCopy()
	policy1WithoutTemplates.Spec.PolicyTemplates = nil

	policy2WithoutTemplates := policy2.DeepCopy()
	policy2WithoutTemplates.Spec.PolicyTemplates = nil

	return common.CompareSpecAndAnnotation(policy1WithoutTemplates, policy2WithoutTemplates)
}
