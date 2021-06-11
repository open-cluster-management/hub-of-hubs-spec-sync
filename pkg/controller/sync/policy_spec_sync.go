// Copyright (c) 2020 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package sync

import (
	"github.com/jackc/pgx/v4/pgxpool"
	policiesv1 "github.com/open-cluster-management/governance-policy-propagator/pkg/apis/policy/v1"
	"github.com/open-cluster-management/governance-policy-propagator/pkg/controller/common"
	ctrl "sigs.k8s.io/controller-runtime"
)

func AddPolicyController(mgr ctrl.Manager, databaseConnectionPool *pgxpool.Pool) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&policiesv1.Policy{}).
		Complete(&PolicyReconciler{genericSpecToDBReconciler{
			client:                 mgr.GetClient(),
			databaseConnectionPool: databaseConnectionPool,
			log:                    ctrl.Log.WithName("policy-spec-syncer"),
			tableName:              "policies",
			finalizerName:          "hub-of-hubs.open-cluster-management.io/policy-cleanup",
			areEqual:               arePoliciesEqual}})
}

type PolicyReconciler struct {
	genericSpecToDBReconciler
}

func (r *PolicyReconciler) Reconcile(request ctrl.Request) (ctrl.Result, error) {
	return r.reconcile(request, &policiesv1.Policy{}, &policiesv1.Policy{})
}

func arePoliciesEqual(instance1, instance2 object) bool {
	policy1 := instance1.(*policiesv1.Policy)
	policy2 := instance2.(*policiesv1.Policy)

	//TODO handle Template comparison later
	policy1WithoutTemplates := policy1.DeepCopy()
	policy1WithoutTemplates.Spec.PolicyTemplates = nil

	policy2WithoutTemplates := policy2.DeepCopy()
	policy2WithoutTemplates.Spec.PolicyTemplates = nil

	return common.CompareSpecAndAnnotation(policy1WithoutTemplates, policy2WithoutTemplates)
}
