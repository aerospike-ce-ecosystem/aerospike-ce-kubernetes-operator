package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	// ClusterPhase reports the current phase of each AerospikeCECluster.
	// Values: 0=Unknown, 1=InProgress, 2=Completed, 3=Error
	ClusterPhase = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "aerospike_ce_cluster_phase",
			Help: "Current phase of the AerospikeCECluster (0=Unknown, 1=InProgress, 2=Completed, 3=Error)",
		},
		[]string{"namespace", "name"},
	)

	// ClusterReadyPods reports the number of ready pods per cluster.
	ClusterReadyPods = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "aerospike_ce_cluster_ready_pods",
			Help: "Number of ready pods in the AerospikeCECluster",
		},
		[]string{"namespace", "name"},
	)

	// ReconcileDuration tracks the duration of reconciliation loops.
	ReconcileDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "aerospike_ce_reconcile_duration_seconds",
			Help:    "Duration of AerospikeCECluster reconciliation in seconds",
			Buckets: prometheus.ExponentialBuckets(0.1, 2, 10), // 0.1s to ~51.2s
		},
		[]string{"namespace", "name"},
	)

	// WarmRestartsTotal counts the number of warm restarts (SIGUSR1) performed.
	WarmRestartsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "aerospike_ce_warm_restarts_total",
			Help: "Total number of warm restarts (SIGUSR1) performed",
		},
		[]string{"namespace", "name"},
	)

	// ColdRestartsTotal counts the number of cold restarts (pod delete) performed.
	ColdRestartsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "aerospike_ce_cold_restarts_total",
			Help: "Total number of cold restarts (pod delete) performed",
		},
		[]string{"namespace", "name"},
	)

	// DynamicConfigUpdatesTotal counts successful dynamic config updates.
	DynamicConfigUpdatesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "aerospike_ce_dynamic_config_updates_total",
			Help: "Total number of successful dynamic config updates via set-config",
		},
		[]string{"namespace", "name"},
	)

	// ACLSyncTotal counts the number of ACL synchronizations performed.
	ACLSyncTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "aerospike_ce_acl_sync_total",
			Help: "Total number of ACL synchronization operations performed",
		},
		[]string{"namespace", "name", "result"},
	)
)

// PhaseToFloat converts a phase string to a float64 for the gauge.
func PhaseToFloat(phase string) float64 {
	switch phase {
	case "InProgress":
		return 1
	case "Completed":
		return 2
	case "Error":
		return 3
	default:
		return 0
	}
}

// CleanupClusterMetrics removes all metrics for a deleted cluster.
func CleanupClusterMetrics(namespace, name string) {
	labels := prometheus.Labels{"namespace": namespace, "name": name}
	ClusterPhase.Delete(labels)
	ClusterReadyPods.Delete(labels)
	ReconcileDuration.Delete(labels)
	WarmRestartsTotal.Delete(labels)
	ColdRestartsTotal.Delete(labels)
	DynamicConfigUpdatesTotal.Delete(labels)

	for _, result := range []string{"success", "error"} {
		ACLSyncTotal.Delete(prometheus.Labels{"namespace": namespace, "name": name, "result": result})
	}
}

func init() {
	metrics.Registry.MustRegister(
		ClusterPhase,
		ClusterReadyPods,
		ReconcileDuration,
		WarmRestartsTotal,
		ColdRestartsTotal,
		DynamicConfigUpdatesTotal,
		ACLSyncTotal,
	)
}
