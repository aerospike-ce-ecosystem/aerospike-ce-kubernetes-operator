//go:build e2e
// +build e2e

/*
Copyright 2026.

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

package e2e

import (
	"fmt"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/ksr/aerospike-ce-kubernetes-operator/test/utils"
)

const (
	aerospikeNS     = "aerospike"
	defaultTimeout  = 3 * time.Minute
	multiNodeTimeout = 5 * time.Minute
)

var _ = Describe("AerospikeCECluster Samples", Ordered, func() {
	var projectDir string

	BeforeAll(func() {
		var err error
		projectDir, err = utils.GetProjectDir()
		Expect(err).NotTo(HaveOccurred())

		By("creating aerospike namespace")
		Expect(utils.CreateNamespaceIfNotExists(aerospikeNS)).To(Succeed())
	})

	AfterAll(func() {
		By("cleaning up all sample clusters")
		for _, name := range []string{"aerospike-ce-basic", "aerospike-ce-3node", "aerospike-ce-multirack", "aerospike-ce-acl"} {
			_ = utils.DeleteAerospikeCluster(name, aerospikeNS)
		}
		By("deleting aerospike namespace")
		_ = utils.DeleteNamespaceIfExists(aerospikeNS)
	})

	Context("Basic single-node cluster", func() {
		const clusterName = "aerospike-ce-basic"

		It("should deploy and reach Completed phase", func() {
			samplePath := filepath.Join(projectDir, "config", "samples", "acko_v1alpha1_aerospikececluster.yaml")

			By("applying the basic sample CR")
			Expect(utils.ApplyFromFile(samplePath)).To(Succeed())

			By("waiting for Completed phase")
			Expect(utils.WaitForClusterPhase(clusterName, aerospikeNS, "Completed", defaultTimeout)).To(Succeed())

			By("verifying 1 pod is running and ready")
			Expect(utils.WaitForPodCount(clusterName, aerospikeNS, 1, defaultTimeout)).To(Succeed())
		})

		It("should create expected Kubernetes resources", func() {
			By("verifying headless service exists")
			Expect(utils.ResourceExists("service", clusterName, aerospikeNS)).To(BeTrue(),
				"headless service should exist")

			By("verifying StatefulSet exists with replicas=1")
			stsNames, err := utils.GetStatefulSetNames(clusterName, aerospikeNS)
			Expect(err).NotTo(HaveOccurred())
			Expect(stsNames).To(HaveLen(1))
			Expect(stsNames[0]).To(Equal(fmt.Sprintf("%s-0", clusterName)))

			By("verifying ConfigMap exists")
			Expect(utils.ResourceExists("configmap", fmt.Sprintf("%s-0-config", clusterName), aerospikeNS)).To(BeTrue(),
				"configmap should exist")
		})

		It("should populate pod status correctly", func() {
			By("verifying status.pods has 1 entry with IsRunningAndReady=true")
			Eventually(func(g Gomega) {
				podStatus, err := utils.GetPodStatusMap(clusterName, aerospikeNS)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(podStatus).To(HaveLen(1))
				for _, ps := range podStatus {
					g.Expect(ps.IsRunningAndReady).To(BeTrue())
					g.Expect(ps.Image).To(Equal("aerospike:ce-8.1.1.1"))
				}
			}, defaultTimeout, 2*time.Second).Should(Succeed())

			By("verifying status.size is 1")
			size, err := utils.GetClusterStatusField(clusterName, aerospikeNS, "{.status.size}")
			Expect(err).NotTo(HaveOccurred())
			Expect(size).To(Equal("1"))
		})
	})

	Context("3-node cluster with PV storage", func() {
		const clusterName = "aerospike-ce-3node"

		It("should deploy 3 nodes and reach Completed phase", func() {
			samplePath := filepath.Join(projectDir, "config", "samples", "aerospike-ce-cluster-3node.yaml")

			By("applying the 3-node sample CR")
			Expect(utils.ApplyFromFile(samplePath)).To(Succeed())

			By("waiting for Completed phase")
			Expect(utils.WaitForClusterPhase(clusterName, aerospikeNS, "Completed", multiNodeTimeout)).To(Succeed())

			By("verifying 3 pods are running and ready")
			Expect(utils.WaitForPodCount(clusterName, aerospikeNS, 3, multiNodeTimeout)).To(Succeed())
		})

		It("should create PVCs for each pod", func() {
			By("verifying PVCs are created")
			Eventually(func(g Gomega) {
				pvcNames, err := utils.GetPVCNames(clusterName, aerospikeNS)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(pvcNames)).To(BeNumerically(">=", 3),
					"should have at least 3 PVCs for 3 pods")
			}, defaultTimeout, 2*time.Second).Should(Succeed())
		})

		It("should report correct status", func() {
			By("verifying status.size is 3")
			size, err := utils.GetClusterStatusField(clusterName, aerospikeNS, "{.status.size}")
			Expect(err).NotTo(HaveOccurred())
			Expect(size).To(Equal("3"))

			By("verifying all pods have correct image in status")
			podStatus, err := utils.GetPodStatusMap(clusterName, aerospikeNS)
			Expect(err).NotTo(HaveOccurred())
			Expect(podStatus).To(HaveLen(3))
			for _, ps := range podStatus {
				Expect(ps.Image).To(Equal("aerospike:ce-8.1.1.1"))
				Expect(ps.IsRunningAndReady).To(BeTrue())
			}
		})
	})

	Context("Multi-rack 6-node cluster", func() {
		const clusterName = "aerospike-ce-multirack"

		It("should deploy 6 nodes across 3 racks and reach Completed phase", func() {
			samplePath := filepath.Join(projectDir, "config", "samples", "aerospike-ce-cluster-multirack.yaml")

			By("applying the multi-rack sample CR")
			Expect(utils.ApplyFromFile(samplePath)).To(Succeed())

			By("waiting for Completed phase")
			Expect(utils.WaitForClusterPhase(clusterName, aerospikeNS, "Completed", multiNodeTimeout)).To(Succeed())

			By("verifying 6 pods are running and ready")
			Expect(utils.WaitForPodCount(clusterName, aerospikeNS, 6, multiNodeTimeout)).To(Succeed())
		})

		It("should create 3 StatefulSets (one per rack)", func() {
			stsNames, err := utils.GetStatefulSetNames(clusterName, aerospikeNS)
			Expect(err).NotTo(HaveOccurred())
			Expect(stsNames).To(HaveLen(3))
			Expect(stsNames).To(ContainElements(
				fmt.Sprintf("%s-1", clusterName),
				fmt.Sprintf("%s-2", clusterName),
				fmt.Sprintf("%s-3", clusterName),
			))
		})

		It("should assign rack labels to pods", func() {
			podNames, err := utils.GetPodNames(clusterName, aerospikeNS)
			Expect(err).NotTo(HaveOccurred())
			Expect(podNames).To(HaveLen(6))

			rackCounts := map[string]int{}
			for _, podName := range podNames {
				rack, err := utils.GetPodLabel(podName, aerospikeNS, "acko.io/rack")
				Expect(err).NotTo(HaveOccurred())
				Expect(rack).NotTo(BeEmpty(), "pod %s should have rack label", podName)
				rackCounts[rack]++
			}

			By("verifying each rack has 2 pods")
			Expect(rackCounts).To(HaveLen(3))
			for rack, count := range rackCounts {
				Expect(count).To(Equal(2), "rack %s should have 2 pods", rack)
			}
		})

		It("should create 3 ConfigMaps (one per rack)", func() {
			for _, rackID := range []int{1, 2, 3} {
				cmName := fmt.Sprintf("%s-%d-config", clusterName, rackID)
				Expect(utils.ResourceExists("configmap", cmName, aerospikeNS)).To(BeTrue(),
					"configmap %s should exist", cmName)
			}
		})

		It("should report correct status.size", func() {
			size, err := utils.GetClusterStatusField(clusterName, aerospikeNS, "{.status.size}")
			Expect(err).NotTo(HaveOccurred())
			Expect(size).To(Equal("6"))
		})
	})

	Context("ACL/Storage sample with cascadeDelete", func() {
		const clusterName = "aerospike-ce-acl"

		It("should deploy 3 nodes and reach Completed phase", func() {
			samplePath := filepath.Join(projectDir, "config", "samples", "aerospike-ce-cluster-acl.yaml")

			By("applying the ACL sample CR")
			Expect(utils.ApplyFromFile(samplePath)).To(Succeed())

			By("waiting for Completed phase")
			Expect(utils.WaitForClusterPhase(clusterName, aerospikeNS, "Completed", multiNodeTimeout)).To(Succeed())

			By("verifying 3 pods are running and ready")
			Expect(utils.WaitForPodCount(clusterName, aerospikeNS, 3, multiNodeTimeout)).To(Succeed())
		})

		It("should have PVCs", func() {
			Eventually(func(g Gomega) {
				pvcNames, err := utils.GetPVCNames(clusterName, aerospikeNS)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(pvcNames)).To(BeNumerically(">=", 3))
			}, defaultTimeout, 2*time.Second).Should(Succeed())
		})

		It("should delete PVCs when cluster is deleted (cascadeDelete)", func() {
			By("deleting the cluster")
			Expect(utils.DeleteAerospikeCluster(clusterName, aerospikeNS)).To(Succeed())

			By("waiting for PVCs to be cleaned up")
			Eventually(func(g Gomega) {
				pvcNames, err := utils.GetPVCNames(clusterName, aerospikeNS)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(pvcNames).To(BeEmpty(), "PVCs should be deleted with cascadeDelete=true")
			}, defaultTimeout, 2*time.Second).Should(Succeed())
		})
	})
})
