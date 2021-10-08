// Copyright (c) 2020 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package controller

import (
	"fmt"

	"github.com/jackc/pgx/v4/pgxpool"
	appsv1 "github.com/open-cluster-management/governance-policy-propagator/pkg/apis/apps/v1"
	policiesv1 "github.com/open-cluster-management/governance-policy-propagator/pkg/apis/policy/v1"
	configv1 "github.com/open-cluster-management/hub-of-hubs-data-types/apis/config/v1"
	cdv1 "github.com/openshift/hive/apis/hive/v1"
	hive "github.com/openshift/hive/apis/scheme"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

// AddToScheme adds all the resources to be processed to the Scheme.
func AddToScheme(s *runtime.Scheme) error {
	//  &scheme.Builder{GroupVersion: cdv1.SchemeGroupVersion}
	schemeBuilders := []*scheme.Builder{policiesv1.SchemeBuilder, appsv1.SchemeBuilder, configv1.SchemeBuilder}

	ocpSchemeBuilders := []*hive.Builder{cdv1.SchemeBuilder}

	for _, schemeBuilder := range schemeBuilders {
		if err := schemeBuilder.AddToScheme(s); err != nil {
			return fmt.Errorf("failed to add scheme: %w", err)
		}
	}

	for _, schemeBuilder := range ocpSchemeBuilders {
		if err := schemeBuilder.AddToScheme(s); err != nil {
			return fmt.Errorf("failed to add scheme: %w", err)
		}
	}

	return nil
}

// AddControllers adds all the controllers to the Manager.
func AddControllers(mgr ctrl.Manager, dbConnectionPool *pgxpool.Pool) error {
	addControllerFunctions := []func(ctrl.Manager, *pgxpool.Pool) error{
		addPolicyController, addPlacementRuleController,
		addPlacementBindingController, addHubOfHubsConfigController,
		addClusterDeploymentController,
	}

	for _, addControllerFunction := range addControllerFunctions {
		if err := addControllerFunction(mgr, dbConnectionPool); err != nil {
			return fmt.Errorf("failed to add controller: %w", err)
		}
	}

	return nil
}
