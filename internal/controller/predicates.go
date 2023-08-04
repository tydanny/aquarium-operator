package controller

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

var AquariumLabelPredicate = predicate.NewPredicateFuncs(func(o client.Object) bool {
	return o.GetLabels()[AppKey] == AquariumValue
})
