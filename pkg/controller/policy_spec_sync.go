// Copyright (c) 2020 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package controller

import (
	"github.com/jackc/pgx/v4/pgxpool"
	policiesv1 "github.com/open-cluster-management/governance-policy-propagator/pkg/apis/policy/v1"
	"github.com/open-cluster-management/governance-policy-propagator/pkg/controller/common"
	ctrl "sigs.k8s.io/controller-runtime"
)

func addPolicyController(mgr ctrl.Manager, databaseConnectionPool *pgxpool.Pool) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&policiesv1.Policy{}).
		Complete(&genericSpecToDBReconciler{
			client:                 mgr.GetClient(),
			databaseConnectionPool: databaseConnectionPool,
			log:                    ctrl.Log.WithName("policy-spec-syncer"),
			tableName:              "policies",
			finalizerName:          "hub-of-hubs.open-cluster-management.io/policy-cleanup",
			createInstance:         func() object { return &policiesv1.Policy{} },
			cleanStatus:            cleanPolicyStatus,
			areEqual:               arePoliciesEqual,
		})
}

func cleanPolicyStatus(instance object) {
	policy, ok := instance.(*policiesv1.Policy)

	if !ok {
		panic("wrong instance passed to cleanConfigStatus: not policiesv1.Policy")
	}

	policy.Status = policiesv1.PolicyStatus{}
}

func arePoliciesEqual(instance1, instance2 object) bool {
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
