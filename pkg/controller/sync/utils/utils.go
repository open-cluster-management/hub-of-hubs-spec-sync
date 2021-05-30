package utils

// from https://book.kubebuilder.io/reference/using-finalizers.html

import (
	policiesv1 "github.com/open-cluster-management/governance-policy-propagator/pkg/apis/policy/v1"
	"github.com/open-cluster-management/governance-policy-propagator/pkg/controller/common"
)

// Helper functions to check and remove string from a slice of strings.
func ContainsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func CompareInstances(instance1, instance2 *policiesv1.Policy) bool {
	//TODO handle Template comparison later
	instance1WithoutTemplates := instance1.DeepCopy()
	instance1WithoutTemplates.Spec.PolicyTemplates = nil

	instance2WithoutTemplates := instance2.DeepCopy()
	instance2WithoutTemplates.Spec.PolicyTemplates = nil

	return common.CompareSpecAndAnnotation(instance1WithoutTemplates, instance2WithoutTemplates)
}
