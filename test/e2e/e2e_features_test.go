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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/ksr/aerospike-ce-kubernetes-operator/test/utils"
)

const featuresNS = "aerospike-features"

// clusterYAML generates an inline AerospikeCECluster YAML for testing.
func clusterYAML(name, ns string, size int32, extraSpec string) string {
	return fmt.Sprintf(`apiVersion: acko.io/v1alpha1
kind: AerospikeCECluster
metadata:
  name: %s
  namespace: %s
spec:
  size: %d
  image: aerospike:ce-8.1.1.1
  aerospikeConfig:
    service:
      cluster-name: %s
      proto-fd-max: 15000
    network:
      service:
        address: any
        port: 3000
      heartbeat:
        mode: mesh
        port: 3002
      fabric:
        address: any
        port: 3001
    namespaces:
      - name: test
        replication-factor: 1
        storage-engine:
          type: memory
          data-size: 1073741824
%s`, name, ns, size, name, extraSpec)
}

var _ = Describe("Enhanced Features", Ordered, func() {

	BeforeAll(func() {
		By("creating features test namespace")
		Expect(utils.CreateNamespaceIfNotExists(featuresNS)).To(Succeed())
	})

	AfterAll(func() {
		By("cleaning up all feature test clusters")
		for _, name := range []string{
			"e2e-metrics", "e2e-podstatus", "e2e-config-change",
			"e2e-scale", "e2e-batch", "e2e-paused", "e2e-pdb",
		} {
			_ = utils.DeleteAerospikeCluster(name, featuresNS)
		}
		By("deleting features test namespace")
		_ = utils.DeleteNamespaceIfExists(featuresNS)
	})

	Context("Prometheus Custom Metrics", func() {
		const clusterName = "e2e-metrics"

		It("should expose custom business metrics after cluster creation", func() {
			By("creating a 1-node cluster")
			yaml := clusterYAML(clusterName, featuresNS, 1, "")
			Expect(utils.ApplyFromStdin(yaml)).To(Succeed())

			By("waiting for Completed phase")
			Expect(utils.WaitForClusterPhase(clusterName, featuresNS, "Completed", defaultTimeout)).To(Succeed())

			By("verifying custom metrics exist in the metrics endpoint")
			Eventually(func(g Gomega) {
				metricsOutput, err := getMetricsOutput()
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(metricsOutput).To(ContainSubstring("aerospike_ce_cluster_phase"),
					"cluster phase metric should be present")
				g.Expect(metricsOutput).To(ContainSubstring("aerospike_ce_cluster_ready_pods"),
					"ready pods metric should be present")
				g.Expect(metricsOutput).To(ContainSubstring("aerospike_ce_reconcile_duration_seconds"),
					"reconcile duration metric should be present")
			}, defaultTimeout, 5*time.Second).Should(Succeed())
		})
	})

	Context("Per-Pod Status with ConfigHash and PodSpecHash", func() {
		const clusterName = "e2e-podstatus"

		It("should populate configHash and podSpecHash in status.pods", func() {
			By("creating a 1-node cluster")
			yaml := clusterYAML(clusterName, featuresNS, 1, "")
			Expect(utils.ApplyFromStdin(yaml)).To(Succeed())

			By("waiting for Completed phase")
			Expect(utils.WaitForClusterPhase(clusterName, featuresNS, "Completed", defaultTimeout)).To(Succeed())

			By("verifying status.pods has configHash and podSpecHash")
			Eventually(func(g Gomega) {
				podStatus, err := utils.GetPodStatusMap(clusterName, featuresNS)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(podStatus).To(HaveLen(1))

				for podName, ps := range podStatus {
					g.Expect(ps.ConfigHash).NotTo(BeEmpty(),
						"configHash should not be empty for pod %s", podName)
					g.Expect(ps.PodSpecHash).NotTo(BeEmpty(),
						"podSpecHash should not be empty for pod %s", podName)
					g.Expect(ps.IsRunningAndReady).To(BeTrue())
				}
			}, defaultTimeout, 2*time.Second).Should(Succeed())
		})

		It("should have matching config hash between pod annotation and status", func() {
			podNames, err := utils.GetPodNames(clusterName, featuresNS)
			Expect(err).NotTo(HaveOccurred())
			Expect(podNames).NotTo(BeEmpty())

			podStatus, err := utils.GetPodStatusMap(clusterName, featuresNS)
			Expect(err).NotTo(HaveOccurred())

			for _, podName := range podNames {
				annotationHash, err := utils.GetPodAnnotation(podName, featuresNS, "acko.io/config-hash")
				Expect(err).NotTo(HaveOccurred())
				Expect(annotationHash).NotTo(BeEmpty())

				if ps, ok := podStatus[podName]; ok {
					Expect(ps.ConfigHash).To(Equal(annotationHash),
						"status configHash should match pod annotation for %s", podName)
				}
			}
		})
	})

	Context("Config Change triggers Rolling Restart", func() {
		const clusterName = "e2e-config-change"

		It("should update pods when config changes", func() {
			By("creating a 2-node cluster")
			yaml := clusterYAML(clusterName, featuresNS, 2, "")
			Expect(utils.ApplyFromStdin(yaml)).To(Succeed())

			By("waiting for Completed phase")
			Expect(utils.WaitForClusterPhase(clusterName, featuresNS, "Completed", multiNodeTimeout)).To(Succeed())
			Expect(utils.WaitForPodCount(clusterName, featuresNS, 2, multiNodeTimeout)).To(Succeed())

			By("recording current configHash values")
			oldStatus, err := utils.GetPodStatusMap(clusterName, featuresNS)
			Expect(err).NotTo(HaveOccurred())
			Expect(oldStatus).To(HaveLen(2))
			oldHashes := map[string]string{}
			for name, ps := range oldStatus {
				oldHashes[name] = ps.ConfigHash
			}

			By("patching proto-fd-max from 15000 to 20000")
			patch := `{"spec":{"aerospikeConfig":{"service":{"proto-fd-max":20000}}}}`
			Expect(utils.PatchClusterSpec(clusterName, featuresNS, patch)).To(Succeed())

			By("waiting for cluster to return to Completed phase")
			// First wait for InProgress (config change triggers reconcile)
			time.Sleep(5 * time.Second) // allow controller to detect the change
			Expect(utils.WaitForClusterPhase(clusterName, featuresNS, "Completed", multiNodeTimeout)).To(Succeed())

			By("verifying configHash changed for all pods")
			Eventually(func(g Gomega) {
				newStatus, err := utils.GetPodStatusMap(clusterName, featuresNS)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(newStatus).To(HaveLen(2))
				for name, ps := range newStatus {
					if oldHash, ok := oldHashes[name]; ok {
						g.Expect(ps.ConfigHash).NotTo(Equal(oldHash),
							"configHash should change for pod %s after config update", name)
					}
				}
			}, defaultTimeout, 2*time.Second).Should(Succeed())
		})
	})

	Context("Scale Up and Down", func() {
		const clusterName = "e2e-scale"

		It("should scale up from 1 to 2 nodes", func() {
			By("creating a 1-node cluster")
			yaml := clusterYAML(clusterName, featuresNS, 1, "")
			Expect(utils.ApplyFromStdin(yaml)).To(Succeed())

			By("waiting for Completed phase with 1 pod")
			Expect(utils.WaitForClusterPhase(clusterName, featuresNS, "Completed", defaultTimeout)).To(Succeed())
			Expect(utils.WaitForPodCount(clusterName, featuresNS, 1, defaultTimeout)).To(Succeed())

			By("scaling up to 2 nodes")
			Expect(utils.PatchClusterSpec(clusterName, featuresNS, `{"spec":{"size":2}}`)).To(Succeed())

			By("waiting for 2 pods to be ready")
			Expect(utils.WaitForClusterPhase(clusterName, featuresNS, "Completed", multiNodeTimeout)).To(Succeed())
			Expect(utils.WaitForPodCount(clusterName, featuresNS, 2, multiNodeTimeout)).To(Succeed())

			By("verifying status.size is 2")
			Eventually(func(g Gomega) {
				size, err := utils.GetClusterStatusField(clusterName, featuresNS, "{.status.size}")
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(size).To(Equal("2"))
			}, defaultTimeout, 2*time.Second).Should(Succeed())
		})

		It("should scale down from 2 to 1 node", func() {
			By("scaling down to 1 node")
			Expect(utils.PatchClusterSpec(clusterName, featuresNS, `{"spec":{"size":1}}`)).To(Succeed())

			By("waiting for Completed phase with 1 pod")
			Expect(utils.WaitForClusterPhase(clusterName, featuresNS, "Completed", multiNodeTimeout)).To(Succeed())
			Expect(utils.WaitForPodCount(clusterName, featuresNS, 1, defaultTimeout)).To(Succeed())

			By("verifying status.size is 1")
			Eventually(func(g Gomega) {
				size, err := utils.GetClusterStatusField(clusterName, featuresNS, "{.status.size}")
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(size).To(Equal("1"))
			}, defaultTimeout, 2*time.Second).Should(Succeed())
		})
	})

	Context("RollingUpdateBatchSize", func() {
		const clusterName = "e2e-batch"

		It("should handle batch rolling restart without errors", func() {
			By("creating a 2-node cluster with batchSize=2")
			yaml := clusterYAML(clusterName, featuresNS, 2, "  rollingUpdateBatchSize: 2")
			Expect(utils.ApplyFromStdin(yaml)).To(Succeed())

			By("waiting for Completed phase")
			Expect(utils.WaitForClusterPhase(clusterName, featuresNS, "Completed", multiNodeTimeout)).To(Succeed())
			Expect(utils.WaitForPodCount(clusterName, featuresNS, 2, multiNodeTimeout)).To(Succeed())

			By("triggering a config change to force restart")
			patch := `{"spec":{"aerospikeConfig":{"service":{"proto-fd-max":18000}}}}`
			Expect(utils.PatchClusterSpec(clusterName, featuresNS, patch)).To(Succeed())

			By("waiting for cluster to return to Completed (batch restart should work)")
			time.Sleep(5 * time.Second)
			Expect(utils.WaitForClusterPhase(clusterName, featuresNS, "Completed", multiNodeTimeout)).To(Succeed())
			Expect(utils.WaitForPodCount(clusterName, featuresNS, 2, multiNodeTimeout)).To(Succeed())
		})
	})

	Context("Paused Cluster", func() {
		const clusterName = "e2e-paused"

		It("should not reconcile config changes while paused", func() {
			By("creating a 1-node cluster")
			yaml := clusterYAML(clusterName, featuresNS, 1, "")
			Expect(utils.ApplyFromStdin(yaml)).To(Succeed())

			By("waiting for Completed phase")
			Expect(utils.WaitForClusterPhase(clusterName, featuresNS, "Completed", defaultTimeout)).To(Succeed())

			By("recording current configHash")
			oldStatus, err := utils.GetPodStatusMap(clusterName, featuresNS)
			Expect(err).NotTo(HaveOccurred())
			var oldHash string
			for _, ps := range oldStatus {
				oldHash = ps.ConfigHash
			}

			By("pausing the cluster")
			Expect(utils.PatchClusterSpec(clusterName, featuresNS, `{"spec":{"paused":true}}`)).To(Succeed())

			By("changing config while paused")
			patch := `{"spec":{"aerospikeConfig":{"service":{"proto-fd-max":25000}}}}`
			Expect(utils.PatchClusterSpec(clusterName, featuresNS, patch)).To(Succeed())

			By("waiting 30 seconds to confirm no reconciliation occurs")
			time.Sleep(30 * time.Second)

			By("verifying configHash has NOT changed (cluster is paused)")
			pausedStatus, err := utils.GetPodStatusMap(clusterName, featuresNS)
			Expect(err).NotTo(HaveOccurred())
			for _, ps := range pausedStatus {
				Expect(ps.ConfigHash).To(Equal(oldHash),
					"configHash should not change while paused")
			}

			By("unpausing the cluster")
			Expect(utils.PatchClusterSpec(clusterName, featuresNS, `{"spec":{"paused":false}}`)).To(Succeed())

			By("waiting for cluster to reconcile and return to Completed")
			Expect(utils.WaitForClusterPhase(clusterName, featuresNS, "Completed", multiNodeTimeout)).To(Succeed())

			By("verifying configHash has changed after unpause")
			Eventually(func(g Gomega) {
				newStatus, err := utils.GetPodStatusMap(clusterName, featuresNS)
				g.Expect(err).NotTo(HaveOccurred())
				for _, ps := range newStatus {
					g.Expect(ps.ConfigHash).NotTo(Equal(oldHash),
						"configHash should change after unpause and config update")
				}
			}, defaultTimeout, 2*time.Second).Should(Succeed())
		})
	})

	Context("PodDisruptionBudget", func() {
		const clusterName = "e2e-pdb"

		It("should create PDB by default and delete when disabled", func() {
			By("creating a 2-node cluster")
			yaml := clusterYAML(clusterName, featuresNS, 2, "")
			Expect(utils.ApplyFromStdin(yaml)).To(Succeed())

			By("waiting for Completed phase")
			Expect(utils.WaitForClusterPhase(clusterName, featuresNS, "Completed", multiNodeTimeout)).To(Succeed())

			By("verifying PDB exists")
			Eventually(func(g Gomega) {
				g.Expect(utils.ResourceExists("pdb", fmt.Sprintf("%s-pdb", clusterName), featuresNS)).To(BeTrue(),
					"PDB should exist for cluster")
			}, defaultTimeout, 2*time.Second).Should(Succeed())

			By("disabling PDB")
			Expect(utils.PatchClusterSpec(clusterName, featuresNS, `{"spec":{"disablePDB":true}}`)).To(Succeed())

			By("verifying PDB is deleted")
			Eventually(func(g Gomega) {
				g.Expect(utils.ResourceExists("pdb", fmt.Sprintf("%s-pdb", clusterName), featuresNS)).To(BeFalse(),
					"PDB should be deleted when disablePDB=true")
			}, defaultTimeout, 2*time.Second).Should(Succeed())
		})
	})
})
