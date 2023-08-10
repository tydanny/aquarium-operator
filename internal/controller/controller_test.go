package controller_test

import (
	"context"
	"reflect"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	funv1alpha1 "github.com/tydanny/aquarium-operator/api/v1alpha1"
)

var _ = Describe("Controller", func() {
	const (
		AquariumName = "test-aquarium"
	)

	Context("When the deployment becomes ready", func() {
		It("should update the aquarium status", func() {
			ctx := context.Background()

			By("Creating an aquarium")
			aquarium := &funv1alpha1.Aquarium{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "fun.tydanny.com/v1alpha1",
					Kind:       reflect.TypeOf(funv1alpha1.Aquarium{}).Name(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      AquariumName,
					Namespace: AquariumNamespace,
				},
				Spec: funv1alpha1.AquariumSpec{
					NumTanks: 2,
					Location: "Atlanta",
				},
			}

			Expect(k8sClient.Create(ctx, aquarium)).Should(Succeed())

			aquariumLookupKey := types.NamespacedName{Name: aquarium.Name, Namespace: AquariumNamespace}
			createdAquarium := &funv1alpha1.Aquarium{}
			Eventually(ctx, func() error {
				return k8sClient.Get(ctx, aquariumLookupKey, createdAquarium)
			}).Should(Succeed())

			By("Checking that the aquarium is unhealthy")
			Consistently(ctx, func() (funv1alpha1.FishHealth, error) {
				if err := k8sClient.Get(ctx, aquariumLookupKey, createdAquarium); err != nil {
					return "", err
				}

				return createdAquarium.Status.FishHealth, nil
			}).Should(Or(Equal(funv1alpha1.Unhealthy), BeEmpty()))

			By("Checking that a deployment is created")
			createdDeployment := &appsv1.Deployment{}
			Eventually(ctx, func() error {
				return k8sClient.Get(ctx, aquariumLookupKey, createdDeployment)
			}).Should(Succeed())
			Expect(*createdDeployment.Spec.Replicas).To(Equal(aquarium.Spec.NumTanks))
			Expect(createdDeployment.Labels).To(HaveKeyWithValue("app", "Aquarium"))

			By("Checking that the aquarium status is updated")
			createdDeployment.Status.Replicas = 2
			createdDeployment.Status.ReadyReplicas = 2
			Expect(k8sClient.Status().Update(ctx, createdDeployment)).To(Succeed())

			Eventually(ctx, func() (funv1alpha1.FishHealth, error) {
				if err := k8sClient.Get(ctx, aquariumLookupKey, createdAquarium); err != nil {
					return "", err
				}

				return createdAquarium.Status.FishHealth, nil
			}).Should(Equal(funv1alpha1.Healthy))
		})
	})
})
