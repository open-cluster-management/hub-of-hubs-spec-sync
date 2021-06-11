// Copyright (c) 2020 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package controller

import (
	"github.com/jackc/pgx/v4/pgxpool"
	appsv1 "github.com/open-cluster-management/governance-policy-propagator/pkg/apis/apps/v1"
	policiesv1 "github.com/open-cluster-management/governance-policy-propagator/pkg/apis/policy/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// AddToScheme adds all Resources to the Scheme.
func AddToScheme(s *runtime.Scheme) error {
	if err := policiesv1.SchemeBuilder.AddToScheme(s); err != nil {
		return err
	}

	if err := appsv1.SchemeBuilder.AddToScheme(s); err != nil {
		return err
	}

	return nil
}

func AddControllers(mgr ctrl.Manager, dbConnectionPool *pgxpool.Pool) error {
	if err := addPolicyController(mgr, dbConnectionPool); err != nil {
		return err
	}

	if err := addPlacementRuleController(mgr, dbConnectionPool); err != nil {
		return err
	}

	if err := addPlacementBindingController(mgr, dbConnectionPool); err != nil {
		return err
	}

	return nil
}
