// Copyright (c) 2020 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package controller

import (
	"fmt"

	"github.com/jackc/pgx/v4/pgxpool"
	policiesv1 "github.com/open-cluster-management/governance-policy-propagator/api/v1"
	configv1 "github.com/open-cluster-management/hub-of-hubs-data-types/apis/config/v1"
	chanv1 "github.com/open-cluster-management/multicloud-operators-channel/pkg/apis/apps/v1"
	appsv1 "github.com/open-cluster-management/multicloud-operators-placementrule/pkg/apis/apps/v1"
	subv1 "github.com/open-cluster-management/multicloud-operators-subscription/pkg/apis/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	applicationv1 "sigs.k8s.io/application/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

// AddToScheme adds all the resources to be processed to the Scheme.
func AddToScheme(sch *runtime.Scheme) error {
	schemeBuilders := []*scheme.Builder{
		policiesv1.SchemeBuilder, appsv1.SchemeBuilder, configv1.SchemeBuilder,
		applicationv1.SchemeBuilder, chanv1.SchemeBuilder, subv1.SchemeBuilder,
	}

	for _, schemeBuilder := range schemeBuilders {
		if err := schemeBuilder.AddToScheme(sch); err != nil {
			return fmt.Errorf("failed to add scheme: %w", err)
		}
	}

	return nil
}

// AddControllers adds all the controllers to the Manager.
func AddControllers(mgr ctrl.Manager, dbConnectionPool *pgxpool.Pool) error {
	addControllerFunctions := []func(ctrl.Manager, *pgxpool.Pool) error{
		addPolicyController, addPlacementRuleController,
		addPlacementBindingController, addHubOfHubsConfigController, addApplicationController,
		addSubscriptionController, addChannelController,
	}

	for _, addControllerFunction := range addControllerFunctions {
		if err := addControllerFunction(mgr, dbConnectionPool); err != nil {
			return fmt.Errorf("failed to add controller: %w", err)
		}
	}

	return nil
}
