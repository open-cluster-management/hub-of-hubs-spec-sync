package controller

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// NamespaceToSkip is the namespace to not reconcile.
const NamespaceToSkip = "open-cluster-management"

func generateNamespacePredicate() predicate.Predicate {
	return predicate.NewPredicateFuncs(func(object client.Object) bool {
		return !(object.GetNamespace() == NamespaceToSkip)
	})
}
