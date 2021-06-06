// Copyright (c) 2020 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package controller

import (
	"github.com/open-cluster-management/hub-of-hubs-spec-syncer/pkg/controller/sync"
)

func init() {
	AddToManagerFuncs = append(AddToManagerFuncs, sync.AddPolicyController, sync.AddPlacementRuleController,
		sync.AddPlacementBindingController)
}
