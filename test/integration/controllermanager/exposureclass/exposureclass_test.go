// Copyright (c) 2022 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package exposureclass_test

import (
	gardencorev1alpha1 "github.com/gardener/gardener/pkg/apis/core/v1alpha1"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"

	. "github.com/gardener/gardener/pkg/utils/test/matchers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("ExposureClass controller test", func() {
	var (
		exposureClass *gardencorev1alpha1.ExposureClass
		shoot         *gardencorev1beta1.Shoot
	)

	BeforeEach(func() {
		exposureClass = &gardencorev1alpha1.ExposureClass{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: testID + "-",
			},
			Handler: "test-exposure-class-handler-name",
			Scheduling: &gardencorev1alpha1.ExposureClassScheduling{
				SeedSelector: &gardencorev1alpha1.SeedSelector{
					LabelSelector: metav1.LabelSelector{
						MatchLabels: map[string]string{
							"test": "foo",
						},
					},
				},
			},
		}

		shoot = &gardencorev1beta1.Shoot{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: testID + "-",
				Namespace:    testNamespace.Name,
			},
			Spec: gardencorev1beta1.ShootSpec{
				CloudProfileName:  "test-cloudprofile",
				SecretBindingName: "my-provider-account",
				Region:            "foo-region",
				Provider: gardencorev1beta1.Provider{
					Type: "test-provider",
					Workers: []gardencorev1beta1.Worker{
						{
							Name:    "cpu-worker",
							Minimum: 2,
							Maximum: 2,
							Machine: gardencorev1beta1.Machine{Type: "large"},
						},
					},
				},
				Kubernetes: gardencorev1beta1.Kubernetes{Version: "1.21.1"},
				Networking: gardencorev1beta1.Networking{Type: "foo-networking"},
			},
		}
	})

	JustBeforeEach(func() {
		By("Create ExposureClass")
		Expect(testClient.Create(ctx, exposureClass)).To(Succeed())
		log.Info("Created ExposureClass for test", "exposureClass", client.ObjectKeyFromObject(exposureClass))

		DeferCleanup(func() {
			By("Delete ExposureClass")
			Expect(testClient.Delete(ctx, exposureClass)).To(Or(Succeed(), BeNotFoundError()))
			Eventually(func() error {
				return testClient.Get(ctx, client.ObjectKeyFromObject(exposureClass), exposureClass)
			}).Should(BeNotFoundError())
		})

		if shoot != nil {
			shoot.Spec.ExposureClassName = pointer.String(exposureClass.Name)

			By("Create Shoot")
			Expect(testClient.Create(ctx, shoot)).To(Succeed())
			log.Info("Created Shoot for test", "shoot", client.ObjectKeyFromObject(shoot))

			DeferCleanup(func() {
				By("Delete Shoot")
				Expect(testClient.Delete(ctx, shoot)).To(Or(Succeed(), BeNotFoundError()))
			})
		}
	})

	Context("no shoot referencing the ExposureClass", func() {
		BeforeEach(func() {
			shoot = nil
		})

		It("should add the finalizer and release it on deletion", func() {
			By("Ensure finalizer got added")
			Eventually(func(g Gomega) {
				g.Expect(testClient.Get(ctx, client.ObjectKeyFromObject(exposureClass), exposureClass)).To(Succeed())
				g.Expect(exposureClass.Finalizers).To(ConsistOf("gardener"))
			}).Should(Succeed())

			By("Delete ExposureClass")
			Expect(testClient.Delete(ctx, exposureClass)).To(Succeed())

			By("Ensure ExposureClass is released")
			Eventually(func() error {
				return testClient.Get(ctx, client.ObjectKeyFromObject(exposureClass), exposureClass)
			}).Should(BeNotFoundError())
		})
	})

	Context("shoots referencing the ExposureClass", func() {
		JustBeforeEach(func() {
			By("Ensure finalizer got added")
			Eventually(func(g Gomega) {
				g.Expect(testClient.Get(ctx, client.ObjectKeyFromObject(exposureClass), exposureClass)).To(Succeed())
				g.Expect(exposureClass.Finalizers).To(ConsistOf("gardener"))
			}).Should(Succeed())

			By("Delete ExposureClass")
			Expect(testClient.Delete(ctx, exposureClass)).To(Succeed())
		})

		It("should add the finalizer and not release it on deletion since there is still referencing shoot", func() {
			By("Ensure ExposureClass is not released")
			Consistently(func() error {
				return testClient.Get(ctx, client.ObjectKeyFromObject(exposureClass), exposureClass)
			}).Should(Succeed())
		})

		It("should add the finalizer and release it on deletion after the shoot got deleted", func() {
			By("Delete Shoot")
			Expect(testClient.Delete(ctx, shoot)).To(Succeed())

			By("Ensure ExposureClass is released")
			Eventually(func() error {
				return testClient.Get(ctx, client.ObjectKeyFromObject(exposureClass), exposureClass)
			}).Should(BeNotFoundError())
		})
	})
})