package agent

import "github.com/prometheus/client_golang/prometheus"

var (
	// Define a counter
	requestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "plakar_agent_requests_total",
			Help: "Total number of processed requests",
		},
		[]string{"method", "status"},
	)

	// Define a gauge
	upGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "plakar_agent_up",
			Help: "Exporter up status",
		},
	)

	disconnectsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "plakar_agent_disconnects_total",
			Help: "Total number of client disconnections",
		},
	)

	backupSuccess = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "plakar_agent_backup_success",
			Help: "Total number of successful backups",
		},
	)

	backupWarning = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "plakar_agent_backup_warning",
			Help: "Total number of successful backups with warnings",
		},
	)

	backupFailure = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "plakar_agent_backup_failure",
			Help: "Total number of failed backups with warnings",
		},
	)

	checkSuccess = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "plakar_agent_check_success",
			Help: "Total number of successful checks",
		},
	)

	checkWarning = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "plakar_agent_check_warning",
			Help: "Total number of successful checks with warnings",
		},
	)

	checkFailure = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "plakar_agent_check_failure",
			Help: "Total number of failed checks with warnings",
		},
	)

	restoreSuccess = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "plakar_agent_restore_success",
			Help: "Total number of successful restores",
		},
	)

	restoreWarning = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "plakar_agent_restore_warning",
			Help: "Total number of successful restores with warnings",
		},
	)

	restoreFailure = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "plakar_agent_restore_failure",
			Help: "Total number of failed restores with warnings",
		},
	)
)

func init() {
	// Register metrics with Prometheus
	//prometheus.MustRegister(requestsTotal)
	//prometheus.MustRegister(upGauge)
	//prometheus.MustRegister(disconnectsTotal)
	prometheus.MustRegister(backupSuccess)
	prometheus.MustRegister(backupWarning)
	prometheus.MustRegister(backupFailure)

	prometheus.MustRegister(checkSuccess)
	prometheus.MustRegister(checkWarning)
	prometheus.MustRegister(checkFailure)

	prometheus.MustRegister(restoreSuccess)
	prometheus.MustRegister(restoreWarning)
	prometheus.MustRegister(restoreFailure)

}

func SuccessInc(task string) {
	switch task {
	case "backup":
		backupSuccess.Inc()
	case "check":
		checkSuccess.Inc()
	case "restore":
		restoreSuccess.Inc()
	}
}

func WarningInc(task string) {
	switch task {
	case "backup":
		backupWarning.Inc()
	case "check":
		checkWarning.Inc()
	case "restore":
		restoreWarning.Inc()
	}
}

func FailureInc(task string) {
	switch task {
	case "backup":
		backupFailure.Inc()
	case "check":
		checkFailure.Inc()
	case "restore":
		restoreFailure.Inc()
	}
}
