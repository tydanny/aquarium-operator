/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"encoding/json"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	funv1alpha1 "github.com/tydanny/aquarium-operator/api/v1alpha1"
)

// AquariumReconciler reconciles a Aquarium object
type AquariumReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=fun.tydanny.com,resources=aquaria,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=fun.tydanny.com,resources=aquaria/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=fun.tydanny.com,resources=aquaria/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile
func (r *AquariumReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("aquarium", req.Name, "ns", req.Namespace)

	// Get our Aquarium CR
	var aquarium funv1alpha1.Aquarium
	if err := r.Get(ctx, req.NamespacedName, &aquarium); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Gather the state of the world
	var aquariumDeploy appsv1.Deployment
	if err := r.Get(ctx, req.NamespacedName, &aquariumDeploy); client.IgnoreNotFound(err) != nil {
		return ctrl.Result{}, err
	}

	// Update Aquarium status
	aquarium.Status = funv1alpha1.AquariumStatus{
		Conditions:    []metav1.Condition{},
		NumTanksReady: aquariumDeploy.Status.AvailableReplicas,
		FishHealth:    funv1alpha1.Unknown,
	}

	if aquariumDeploy.Status.ReadyReplicas == aquarium.Spec.NumTanks {
		aquarium.Status.FishHealth = funv1alpha1.Healthy
	} else {
		aquarium.Status.FishHealth = funv1alpha1.Unhealthy
	}

	// If we fail to update status don't requeue for reconcile.
	// We do this so that we can get to the desired state step.
	if err := r.Status().Update(ctx, &aquarium); err != nil {
		log.Error(err, "failed to update aquarium status")
	}

	// Reach the Aquariums desired state
	aquariumNamespace := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: aquarium.Spec.Location,
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion:         aquarium.APIVersion,
				Kind:               aquarium.Kind,
				Name:               aquarium.Name,
				UID:                aquarium.UID,
				Controller:         pointer.Bool(true),
				BlockOwnerDeletion: pointer.Bool(true),
			}},
			Labels: map[string]string{
				AppKey: AquariumValue,
			},
		},
	}

	if err := r.Create(ctx, &aquariumNamespace); client.IgnoreAlreadyExists(err) != nil {
		return ctrl.Result{}, err
	}

	desiredDeploy := newDeployment(&aquarium)

	if err := r.Apply(ctx, desiredDeploy); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AquariumReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&funv1alpha1.Aquarium{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&appsv1.Deployment{}, builder.WithPredicates(AquariumLabelPredicate)).
		Owns(&corev1.Namespace{}, builder.WithPredicates(AquariumLabelPredicate)).
		Complete(r)
}

func newDeployment(aquarium *funv1alpha1.Aquarium) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      aquarium.Name,
			Namespace: aquarium.Spec.Location,
			Labels: map[string]string{
				AppKey: AquariumValue,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: pointer.Int32(aquarium.Spec.NumTanks),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "Aquarium",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "Aquarium",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:    "aquarium",
						Image:   aquarium.Spec.Image,
						Command: []string{"sleep", "10000"},
					}},
				},
			},
		},
	}
}

// Apply uses server side apply to create Kubernetes objects
func (r *AquariumReconciler) Apply(ctx context.Context, in client.Object, opts ...client.PatchOption) error {
	unstructured, err := getUnstructuredFromObject(r.Scheme, in)
	if err != nil {
		return fmt.Errorf("failed to get unstructured from object: %w", err)
	}

	// In K8s v1.25, server-side apply validation is enabled. The status field is not allowed when patching a resource.
	delete(unstructured.Object, "status")
	return r.Patch(ctx, unstructured, client.Apply, client.ForceOwnership, client.FieldOwner("aquarium-operator"))
}

func getUnstructuredFromObject(scheme *runtime.Scheme, in client.Object) (*unstructured.Unstructured, error) {
	gvk, err := apiutil.GVKForObject(in, scheme)
	if err != nil {
		return nil, err
	}

	js, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}

	var data map[string]interface{}
	if err := json.Unmarshal(js, &data); err != nil {
		return nil, err
	}

	u := &unstructured.Unstructured{Object: data}
	u.SetGroupVersionKind(gvk)

	return u, nil
}
