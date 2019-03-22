package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	puppetdb "github.com/negast/go-puppetdb"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	yaml "gopkg.in/yaml.v2"
)

var (
	addr = flag.String("listen-address", ":8156", "The address to listen on for HTTP requests.")
	conf = flag.String("conf", "puppetdb-facts.yaml", "The path to the config file.")
	endP = flag.String("endpoint", "/metrics", "The path to display the metrics on default  /metrics")
)

var puppetDBGuage = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "puppetdb_connection_up",
		Help: "The check to see if the client settings are ok to connect with",
	},
)

var factGuage = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "puppetdb_facts_quage",
		Help: "Automated gauge for the specified fact",
	},
	[]string{"fact", "puppet_environment", "node"},
)

var factNodeGuage = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "puppetdb_facts_node_quage",
		Help: "Displays the nodes holding the value for the total gauge",
	},
	[]string{"fact", "puppet_environment", "node", "value"},
)

var factTotal = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "puppetdb_facts_total",
		Help: "Automated count for the specified fact",
	},
	[]string{"fact", "value"},
)

var statusTotal = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "puppetdb_status_total",
		Help: "Automated count for the specified report status",
	},
	[]string{"status", "puppet_environment"},
)

var statusNodesGuage = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "puppetdb_status_node_guage",
		Help: "Automated count for the report status per node",
	},
	[]string{"status", "value", "puppet_environment", "node"},
)

var resourcesNodeGuage = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "puppetdb_report_resources_node_guage",
		Help: "Automated count for the report resources per node",
	},
	[]string{"name", "puppet_environment", "node"},
)

var timeNodeGuage = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "puppetdb_report_time_node_guage",
		Help: "Automated count for the report time per node",
	},
	[]string{"name", "puppet_environment", "node"},
)

var eventNodeGuage = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "puppetdb_report_event_node_guage",
		Help: "Automated count for the report time per node",
	},
	[]string{"name", "puppet_environment", "node"},
)

var timeLastReportNodeGuage = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "puppetdb_last_report_time_node_guage",
		Help: "Automated guage for the last report time of a node",
	},
	[]string{"puppet_environment", "node"},
)

var masterHttpGuage = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "puppet_master_http_guage",
		Help: "Http stats for the master",
	},
	[]string{"master", "route", "type"},
)

var masterHttpClientGuage = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "puppet_master_http_client_guage",
		Help: "Http client stats for the master",
	},
	[]string{"master", "metric_name", "type", "metric_id"},
)

var masterJVMGuage = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "puppet_master_jvm_guage",
		Help: "jvm stats for the master",
	},
	[]string{"master", "metric"},
)

var masterMemoryGuage = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "puppet_master_memory_guage",
		Help: "memory stats for the master both heap and non heap memory",
	},
	[]string{"master", "is_heap", "type"},
)

var masterGCCount = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "puppet_master_gc_count",
		Help: "GC count for the master",
	},
	[]string{"master", "ps_type"},
)

var masterGCTime = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "puppet_master_gc_time",
		Help: "GC time in ms for the master",
	},
	[]string{"master", "ps_type"},
)

var masterGCDuration = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "puppet_master_gc_duration",
		Help: "jDuration since the last info in ms",
	},
	[]string{"master", "ps_type"},
)

var masterFile = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "puppet_master_file_guage",
		Help: "file descriptors count",
	},
	[]string{"master", "name"},
)

var masterThread = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "puppet_master_thread_guage",
		Help: "Thread count for the puppet master",
	},
	[]string{"master", "name"},
)

var masterRuby = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "puppet_master_jruby_guage",
		Help: "Thread count for the puppet master",
	},
	[]string{"master", "name"},
)

var masterBorrowedInstance = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "puppet_master_borrowed_instance_time",
		Help: " count for the puppet master",
	},
	[]string{"master", "uri", "method", "routeId"},
)

var masterBorrowedInstanceDuration = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "puppet_master_borrowed_instance_duration_ms",
		Help: "Borrowed instance metrics for the puppet master",
	},
	[]string{"master", "uri", "method", "routeId"},
)

var masterFunction = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "puppet_master_profiler_function_guage",
		Help: "Profiler function metrics for the puppet master",
	},
	[]string{"master", "function", "type"},
)

var masterResource = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "puppet_master_profiler_resource_guage",
		Help: "Profiler resource metrics for the puppet master",
	},
	[]string{"master", "resource", "type"},
)

var masterCatalog = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "puppet_master_profiler_catalog_guage",
		Help: "Profiler catalog metrics for the puppet master",
	},
	[]string{"master", "metric", "type"},
)

var masterPuppetdb = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "puppet_master_profiler_puppetdb_guage",
		Help: "Profiler puppetdb metrics for the puppet master",
	},
	[]string{"master", "metric", "type"},
)

var masterState = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "puppet_master_state_guage",
		Help: "Profiler puppetdb metrics for the puppet master",
	},
	[]string{"master", "service_version", "service_state", "detail_level"},
)

func init() {
	// Register the summary and the histogram with Prometheus's default registry.
	prometheus.MustRegister(factGuage)
	prometheus.MustRegister(puppetDBGuage)
	prometheus.MustRegister(factTotal)
	prometheus.MustRegister(factNodeGuage)
	prometheus.MustRegister(statusTotal)
	prometheus.MustRegister(statusNodesGuage)
	prometheus.MustRegister(resourcesNodeGuage)
	prometheus.MustRegister(timeNodeGuage)
	prometheus.MustRegister(eventNodeGuage)
	prometheus.MustRegister(timeLastReportNodeGuage)

	prometheus.MustRegister(masterHttpGuage)
	prometheus.MustRegister(masterHttpClientGuage)

	prometheus.MustRegister(masterGCCount)
	prometheus.MustRegister(masterGCDuration)
	prometheus.MustRegister(masterGCTime)
	prometheus.MustRegister(masterJVMGuage)
	prometheus.MustRegister(masterMemoryGuage)
	prometheus.MustRegister(masterFile)
	prometheus.MustRegister(masterThread)

	prometheus.MustRegister(masterRuby)
	prometheus.MustRegister(masterBorrowedInstance)
	prometheus.MustRegister(masterBorrowedInstanceDuration)

	prometheus.MustRegister(masterResource)
	prometheus.MustRegister(masterFunction)
	prometheus.MustRegister(masterCatalog)
	prometheus.MustRegister(masterPuppetdb)

	prometheus.MustRegister(masterState)

}

func getBaseMetric(fact string, c *puppetdb.Client) ([]puppetdb.FactJSON, error) {
	facts, err := c.FactPerNode(fact)
	if err != nil {
		log.Print(err)
		puppetDBGuage.Set(0)
	} else {
		puppetDBGuage.Set(1)
	}
	return facts, err
}

func evalMetric(facts []puppetdb.FactJSON, nodes bool) {
	if len(facts) > 0 {
		value := facts[0].Value.Data()
		check := reflect.TypeOf(value).String()
		switch check {
		case "int":
			addGaugeMetricfactsIntOrFloat(facts)
		case "string":
			addGaugeMetricfactsString(facts, nodes)
		case "float64":
			addGaugeMetricfactsIntOrFloat(facts)
		default:
			fmt.Println("Skipping fact because not of type int, float or string")
		}
	}

}

func evalMetricPath(facts []puppetdb.FactJSON, path string, nodes bool) {
	if len(facts) > 0 {
		value := facts[0].Value.Path(path).Data()
		check := reflect.TypeOf(value).String()
		switch check {
		case "int":
			addGaugeMetricfactsIntOrFloatPath(facts, path)
		case "string":
			addGaugeMetricfactsStringPath(facts, path, nodes)
		case "float64":
			addGaugeMetricfactsIntOrFloatPath(facts, path)
		default:
			fmt.Println("Skipping fact because not of type int, float or string")
		}
	}

}

func addGaugeMetricfactsIntOrFloat(facts []puppetdb.FactJSON) {
	for _, fact := range facts {
		value := fact.Value.Data().(float64)
		factGuage.WithLabelValues(fact.Name, fact.Environment, fact.CertName).Set(value)
	}
}

func addGaugeMetricfactsString(facts []puppetdb.FactJSON, nodes bool) {
	for _, fact := range facts {
		value := fact.Value.Data().(string)
		factTotal.WithLabelValues(fact.Name, value).Inc()
		if nodes {
			factNodeGuage.WithLabelValues(fact.Name, fact.Environment, fact.CertName, value).Set(1)
		}
	}
}

func addLastReportTimeMetric(node puppetdb.NodeJSON) {
	layout := "2006-01-02T15:04:05.000Z"
	t, err := time.Parse(layout, node.ReportTimestamp)
	if err != nil {
		fmt.Println(err)
	}
	duration := time.Since(t).Seconds()
	timeLastReportNodeGuage.WithLabelValues(node.ReportEnvironment, node.Certname).Set(duration)

}

func addGaugeMetricStatusString(reports []puppetdb.ReportJSON, nodes bool, node puppetdb.NodeJSON, statusArr map[string]map[string]int) {
	for _, report := range reports {
		failed := 0.0
		cor_changes := 0.0
		changes := 0.0

		for _, metric := range report.Metrics.Data {
			if metric.Category == "time" {
				timeNodeGuage.WithLabelValues(metric.Name, report.Environment, report.CertName).Set(metric.Value)
			} else if metric.Category == "events" {
				eventNodeGuage.WithLabelValues(metric.Name, report.Environment, report.CertName).Set(metric.Value)
			} else if metric.Category == "resources" {
				resourcesNodeGuage.WithLabelValues(metric.Name, report.Environment, report.CertName).Set(metric.Value)
				if metric.Name == "failed" {
					failed = metric.Value
				}
				if metric.Name == "changed" {
					changes = metric.Value
				}
				if metric.Name == "corrective_change" {
					cor_changes = metric.Value
				}
			}
		}

		stat := node.LatestReportStatus
		if stat == "changed" {
			if cor_changes > 0.0 {
				stat = "corrective_changes"
			}
		}

		if _, ok := statusArr["All"]; ok {
			if _, ok := statusArr["All"][stat]; ok {
				statusArr["All"][stat] = statusArr["All"][stat] + 1
			} else {
				statusArr["All"][stat] = 1
			}
		} else {
			statusArr["All"] = map[string]int{
				stat: 1,
			}
		}
		if _, ok := statusArr[node.ReportEnvironment]; ok {
			if _, ok := statusArr[node.ReportEnvironment][stat]; ok {
				statusArr[node.ReportEnvironment][stat] = statusArr[node.ReportEnvironment][stat] + 1
			} else {
				statusArr[node.ReportEnvironment][stat] = 1
			}
		} else {
			statusArr[node.ReportEnvironment] = map[string]int{
				stat: 1,
			}
		}

		if nodes {
			if node.LatestReportStatus == "unchanged" {
				statusNodesGuage.WithLabelValues(report.Status, "1", report.Environment, report.CertName).Inc()
			}
			if node.LatestReportStatus == "failed" {
				value := strconv.Itoa(int(failed))
				statusNodesGuage.WithLabelValues(report.Status, value, report.Environment, report.CertName).Inc()
			}
			if node.LatestReportStatus == "changed" {
				if cor_changes > 0 {
					value := strconv.Itoa(int(cor_changes))
					statusNodesGuage.WithLabelValues("corrective_change", value, report.Environment, report.CertName).Inc()
				}
				if changes > 0 {
					value := strconv.Itoa(int(changes))
					statusNodesGuage.WithLabelValues("changed", value, report.Environment, report.CertName).Inc()
				}
			}
		}
	}
}

func addGaugeMetricfactsIntOrFloatPath(facts []puppetdb.FactJSON, path string) {
	for _, fact := range facts {
		value := fact.Value.Path(path).Data().(float64)
		factGuage.WithLabelValues(fact.Name+"."+path, fact.Environment, fact.CertName).Set(value)
	}
}

func addGaugeMetricfactsStringPath(facts []puppetdb.FactJSON, path string, nodes bool) {
	for _, fact := range facts {
		value := fact.Value.Path(path).Data().(string)
		factTotal.WithLabelValues(fact.Name+"."+path, value).Inc()
		if nodes {
			factNodeGuage.WithLabelValues(fact.Name+"."+path, fact.Environment, fact.CertName, value).Set(1)
		}
	}
}

type Conf struct {
	Facts        []string `yaml:"facts"`
	Nodes        bool     `yaml:"nodes"`
	Host         string   `yaml:"host"`
	Port         int      `yaml:"port"`
	MasterEnable bool     `yaml:"masterEnable"`
	MasterHost   string   `yaml:"masterHost"`
	MasterPort   int      `yaml:"masterPort"`
	SSL          bool     `yaml:"ssl"`
	Key          string   `yaml:"key"`
	Ca           string   `yaml:"ca"`
	Cert         string   `yaml:"cert"`
	Interval     int      `yaml:"interval"`
	Debug        bool     `yaml:"debug"`
	Timeout      int      `yaml:"timeout"`
}

func (c *Conf) getConf(configFile string) *Conf {

	yamlFile, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	return c
}

func GenerateFactsMetrics(facts []string, c *puppetdb.Client, nodes bool, debug bool) {
	if debug {
		log.Print("Resetting facts interfaces.")
	}
	factTotal.Reset()
	factGuage.Reset()
	factNodeGuage.Reset()
	if debug {
		log.Print("Done resetting facts interfaces.")
	}
	for _, fact := range facts {
		// TODO more fact debugging
		fA := strings.Split(fact, ".")
		if len(fA) == 1 {
			facts2, _ := getBaseMetric(fA[0], c)
			evalMetric(facts2, nodes)
		} else if len(fA) > 1 {
			facts2, _ := getBaseMetric(fA[0], c)
			pathA := append(fA[:0], fA[0+1:]...)
			path := strings.Join(pathA, ".")
			evalMetricPath(facts2, path, nodes)
		}
	}
}

func GenerateReportsMetrics(c *puppetdb.Client, nodes bool, debug bool, timeout int) {
	if debug {
		log.Print("Retrieving nodes.")
	}
	nodesArr, err := c.Nodes()

	if err != nil {
		log.Print(err)
	}
	if debug {
		i := len(nodesArr)
		log.Printf("Nodes collected found %d nodes", i)
		log.Print("Ressting status and node metrics interfaces.")
	}
	// Reset nodes values
	statusNodesGuage.Reset()
	resourcesNodeGuage.Reset()
	timeNodeGuage.Reset()
	eventNodeGuage.Reset()
	if debug {
		log.Print("Done resetting the interfaces")
	}

	var statusArr map[string]map[string]int = map[string]map[string]int{}

	for _, node := range nodesArr {
		// set the report time
		addLastReportTimeMetric(node)

		// eval time
		layout := "2006-01-02T15:04:05.000Z"
		t, err := time.Parse(layout, node.ReportTimestamp)
		if err != nil {
			fmt.Println(err)
		}
		duration := time.Since(t)
		timeout := float64(timeout)
		// if timeout reached we do not parse this node
		if duration.Seconds() > timeout {
			if debug {
				log.Printf("Node % has reached the timeout and is in unresponsive state latest report was %s", node.Certname, node.ReportTimestamp)
			}
			timeoutString := strconv.Itoa(int(duration.Seconds()))

			if nodes {
				statusNodesGuage.WithLabelValues("unresponsive", timeoutString, node.ReportEnvironment, node.Certname).Inc()
			}

			// set the unresponsive state
			if _, ok := statusArr["All"]; ok {
				if _, ok := statusArr["All"]["unresponsive"]; ok {
					statusArr["All"]["unresponsive"] = statusArr["All"]["unresponsive"] + 1
				} else {
					statusArr["All"]["unresponsive"] = 1
				}
			} else {
				statusArr["All"] = map[string]int{
					"unresponsive": 1,
				}
			}
			if _, ok := statusArr[node.ReportEnvironment]; ok {
				if _, ok := statusArr[node.ReportEnvironment]["unresponsive"]; ok {
					statusArr[node.ReportEnvironment]["unresponsive"] = statusArr[node.ReportEnvironment]["unresponsive"] + 1
				} else {
					statusArr[node.ReportEnvironment]["unresponsive"] = 1
				}
			} else {
				statusArr[node.ReportEnvironment] = map[string]int{
					"unresponsive": 1,
				}
			}

			// if node is okay we parse it
		} else {
			res, err := c.ReportByHash(node.LatestReportHash)
			if debug {
				log.Printf("Retrieving report for node: %s  of environment: %s latest report hash is: %s latest report status is %s latest report time is %s", node.Certname, node.ReportEnvironment, node.LatestReportHash, node.LatestReportStatus, node.ReportTimestamp)
			}
			addGaugeMetricStatusString(res, nodes, node, statusArr)
			if err != nil {
				log.Print(err)
			}

		}

	}
	statusTotal.Reset()
	for key, _ := range statusArr {
		for key2, value2 := range statusArr[key] {
			statusTotal.WithLabelValues(key2, key).Set(float64(value2))
		}
	}

}

func GeneratePuppetMasterMetrics(cl2 *puppetdb.ClientMaster, host string, debug bool) {
	m, err := cl2.Master()
	if err != nil {
		log.Println(err.Error())
	} else {
		generateHttpMetrics(&m, host, debug)
		masterState.WithLabelValues(host, m.Version, m.State, m.DetailLevel).Set(1)
	}

	j, err := cl2.Service()
	if err != nil {
		log.Println(err.Error())
	} else {
		generateJVMMetrics(&j, host, debug)
	}
	r, err := cl2.Jruby()
	if err != nil {
		log.Println(err.Error())
	} else {
		generateJrubyMetrics(&r, host, debug)
	}
	p, err := cl2.Profiler()
	if err != nil {
		log.Println(err.Error())
	} else {
		generateProfileMetrics(&p, host, debug)
	}
}

func generateHttpMetrics(master *puppetdb.MasterMetrics, host string, debug bool) {
	for _, metric := range *master.Status.Experimental.HttpMetrics {
		masterHttpGuage.WithLabelValues(host, metric.RouteId, "aggregate").Set(float64(metric.Aggregate))
		masterHttpGuage.WithLabelValues(host, metric.RouteId, "count").Set(float64(metric.Count))
		masterHttpGuage.WithLabelValues(host, metric.RouteId, "mean").Set(float64(metric.Mean))
	}

	for _, metric := range *master.Status.Experimental.HttpClientMetrics {
		masterHttpClientGuage.WithLabelValues(host, metric.MetricName, "aggregate", strings.Join(*metric.MetricId, ",")).Set(float64(metric.Aggregate))
		masterHttpClientGuage.WithLabelValues(host, metric.MetricName, "count", strings.Join(*metric.MetricId, ",")).Set(float64(metric.Count))
		masterHttpClientGuage.WithLabelValues(host, metric.MetricName, "mean", strings.Join(*metric.MetricId, ",")).Set(float64(metric.Mean))

	}
}

func generateJVMMetrics(jvm *puppetdb.ServiceMetrics, host string, debug bool) {
	//prometheus.MustRegister(masterFile)
	masterFile.WithLabelValues(host, "used").Set(float64(jvm.Status.Experimental.JVMMetrics.FileDescriptors.Used))
	masterFile.WithLabelValues(host, "max").Set(float64(jvm.Status.Experimental.JVMMetrics.FileDescriptors.Max))

	// prometheus.MustRegister(masterThread)
	masterThread.WithLabelValues(host, "normal").Set(float64(jvm.Status.Experimental.JVMMetrics.Threading.ThreadCount))
	masterThread.WithLabelValues(host, "peak").Set(float64(jvm.Status.Experimental.JVMMetrics.Threading.PeakThreadCount))

	// prometheus.MustRegister(masterJVMGuage)
	masterJVMGuage.WithLabelValues(host, "cpu_usage").Set(jvm.Status.Experimental.JVMMetrics.CpuUsage)
	masterJVMGuage.WithLabelValues(host, "up_time_ms").Set(float64(jvm.Status.Experimental.JVMMetrics.UptimeMs))
	masterJVMGuage.WithLabelValues(host, "gc_cpu_usage").Set(jvm.Status.Experimental.JVMMetrics.GCCpuUsage)
	masterJVMGuage.WithLabelValues(host, "start_time_ms").Set(float64(jvm.Status.Experimental.JVMMetrics.StartTimeMs))

	// prometheus.MustRegister(masterMemoryGuage)
	masterMemoryGuage.WithLabelValues(host, "true", "commited").Set(float64(jvm.Status.Experimental.JVMMetrics.HeapMemory.Committed))
	masterMemoryGuage.WithLabelValues(host, "true", "init").Set(float64(jvm.Status.Experimental.JVMMetrics.HeapMemory.Init))
	masterMemoryGuage.WithLabelValues(host, "true", "max").Set(float64(jvm.Status.Experimental.JVMMetrics.HeapMemory.Max))
	masterMemoryGuage.WithLabelValues(host, "true", "used").Set(float64(jvm.Status.Experimental.JVMMetrics.HeapMemory.Used))

	masterMemoryGuage.WithLabelValues(host, "false", "commited").Set(float64(jvm.Status.Experimental.JVMMetrics.NonHeapMemory.Committed))
	masterMemoryGuage.WithLabelValues(host, "false", "init").Set(float64(jvm.Status.Experimental.JVMMetrics.NonHeapMemory.Init))
	masterMemoryGuage.WithLabelValues(host, "false", "max").Set(float64(jvm.Status.Experimental.JVMMetrics.NonHeapMemory.Max))
	masterMemoryGuage.WithLabelValues(host, "false", "used").Set(float64(jvm.Status.Experimental.JVMMetrics.NonHeapMemory.Used))

	// prometheus.MustRegister(masterGCCount)
	masterGCCount.WithLabelValues(host, "scavenge").Set(float64(jvm.Status.Experimental.JVMMetrics.GCStats.PSScavenge.Count))
	masterGCCount.WithLabelValues(host, "marksweep").Set(float64(jvm.Status.Experimental.JVMMetrics.GCStats.PSSweep.Count))

	// prometheus.MustRegister(masterGCDuration)
	masterGCDuration.WithLabelValues(host, "scavenge").Set(float64(jvm.Status.Experimental.JVMMetrics.GCStats.PSScavenge.LastGCInfo.DurationMs))
	masterGCDuration.WithLabelValues(host, "marksweep").Set(float64(jvm.Status.Experimental.JVMMetrics.GCStats.PSSweep.LastGCInfo.DurationMs))

	// prometheus.MustRegister(masterGCTime)
	masterGCTime.WithLabelValues(host, "scavenge").Set(float64(jvm.Status.Experimental.JVMMetrics.GCStats.PSScavenge.TotalTimeMs))
	masterGCTime.WithLabelValues(host, "marksweep").Set(float64(jvm.Status.Experimental.JVMMetrics.GCStats.PSSweep.TotalTimeMs))
}

func generateJrubyMetrics(r *puppetdb.JrubyMetrics, host string, debug bool) {
	// prometheus.MustRegister(masterRuby)
	masterRuby.WithLabelValues(host, "average_lock_wait_time").Set(float64(r.Status.Experimental.Metrics.AverageLockWaitTime))
	masterRuby.WithLabelValues(host, "num_free_jrubies").Set(float64(r.Status.Experimental.Metrics.NumFreeJrubies))
	masterRuby.WithLabelValues(host, "borrow_count").Set(float64(r.Status.Experimental.Metrics.BorrowCount))
	masterRuby.WithLabelValues(host, "borrow_timeout_count").Set(float64(r.Status.Experimental.Metrics.BorrowTimeoutCount))
	masterRuby.WithLabelValues(host, "avg_requested_jrubies").Set(float64(r.Status.Experimental.Metrics.AverageRequestedJrubies))

	masterRuby.WithLabelValues(host, "return_count").Set(float64(r.Status.Experimental.Metrics.ReturnCount))
	masterRuby.WithLabelValues(host, "borrow_retry_count").Set(float64(r.Status.Experimental.Metrics.BorrowRetryCount))
	masterRuby.WithLabelValues(host, "average_borrow_time").Set(float64(r.Status.Experimental.Metrics.AverageBorrowTime))
	masterRuby.WithLabelValues(host, "num_jrubies").Set(float64(r.Status.Experimental.Metrics.NumJrubies))
	masterRuby.WithLabelValues(host, "requested_count").Set(float64(r.Status.Experimental.Metrics.RequestedCount))

	masterRuby.WithLabelValues(host, "average_lock_held_time").Set(float64(r.Status.Experimental.Metrics.AverageLockHeldTime))
	masterRuby.WithLabelValues(host, "queue_limit_hit_count").Set(float64(r.Status.Experimental.Metrics.QueueLimitHitCount))
	masterRuby.WithLabelValues(host, "average_free_jrubies").Set(float64(r.Status.Experimental.Metrics.AverageFreeJrubies))
	masterRuby.WithLabelValues(host, "num_pool_locks").Set(float64(r.Status.Experimental.Metrics.NumPoolLocks))
	masterRuby.WithLabelValues(host, "average_wait_time").Set(float64(r.Status.Experimental.Metrics.AverageWaitTime))
	masterRuby.WithLabelValues(host, "queue_limit-hit_rate").Set(float64(r.Status.Experimental.Metrics.QueueLimitHitRate))

	// prometheus.MustRegister(masterBorrowedInstance)
	masterBorrowedInstance.Reset()
	masterBorrowedInstanceDuration.Reset()

	for _, metric := range *r.Status.Experimental.Metrics.BorrowedInstances {
		masterBorrowedInstance.WithLabelValues(host, metric.Reason.Request.Uri,
			metric.Reason.Request.Method, metric.Reason.Request.RouteId).Set(float64(metric.Time))
		masterBorrowedInstanceDuration.WithLabelValues(host, metric.Reason.Request.Uri,
			metric.Reason.Request.Method, metric.Reason.Request.RouteId).Set(float64(metric.DurationMilis))
	}
}

func generateProfileMetrics(p *puppetdb.Profiler, host string, debug bool) {
	// prometheus.MustRegister(masterResource)
	//masterResource.Reset()
	for _, metric := range *p.Status.Experimental.ResourceMetrics {
		masterResource.WithLabelValues(host, metric.Resource, "count").Set(float64(metric.Count))
		masterResource.WithLabelValues(host, metric.Resource, "mean").Set(float64(metric.Mean))
		masterResource.WithLabelValues(host, metric.Resource, "aggregate").Set(float64(metric.Aggregate))

	}
	//prometheus.MustRegister(masterFunction)
	for _, metric := range *p.Status.Experimental.FunctionMetrics {
		masterFunction.WithLabelValues(host, metric.Function, "count").Set(float64(metric.Count))
		masterFunction.WithLabelValues(host, metric.Function, "mean").Set(float64(metric.Mean))
		masterFunction.WithLabelValues(host, metric.Function, "aggregate").Set(float64(metric.Aggregate))

	}
	//prometheus.MustRegister(masterCatalog)
	for _, metric := range *p.Status.Experimental.CatalogMetrics {
		masterCatalog.WithLabelValues(host, metric.Metric, "count").Set(float64(metric.Count))
		masterCatalog.WithLabelValues(host, metric.Metric, "mean").Set(float64(metric.Mean))
		masterCatalog.WithLabelValues(host, metric.Metric, "aggregate").Set(float64(metric.Aggregate))

	}
	//prometheus.MustRegister(masterPuppetdb)
	for _, metric := range *p.Status.Experimental.PuppetdbMetrics {
		masterPuppetdb.WithLabelValues(host, metric.Metric, "count").Set(float64(metric.Count))
		masterPuppetdb.WithLabelValues(host, metric.Metric, "mean").Set(float64(metric.Mean))
		masterPuppetdb.WithLabelValues(host, metric.Metric, "aggregate").Set(float64(metric.Aggregate))

	}
}

func main() {
	flag.Parse()

	var c Conf
	c.getConf(*conf)
	if c.Host == "" {
		c.Host = "localhost"
	}
	if c.MasterHost == "" {
		c.MasterHost = c.Host
	}
	if c.Port == 0 {
		c.Port = 8080
	}
	if c.MasterPort == 0 {
		c.MasterPort = 8140
	}
	if c.Timeout == 0 {
		c.Timeout = 3600
	}

	//c.Debug = true
	if c.SSL {
		if c.Debug {
			log.Print("SSL was configured continuing with ssl settings on https.")
		}
		go func() {
			for {
				cl := puppetdb.NewClientSSL(c.Host, c.Port, c.Key, c.Cert, c.Ca, false)
				GenerateFactsMetrics(c.Facts, cl, c.Nodes, c.Debug)
				GenerateReportsMetrics(cl, c.Nodes, c.Debug, c.Timeout)
				i := time.Duration(15)
				if c.Interval != 0 {
					i = time.Duration(c.Interval)
				}
				if c.MasterEnable {
					cl2 := puppetdb.NewClientSSLMaster(c.MasterHost, c.MasterPort, c.Key, c.Cert, c.Ca, c.Debug)
					_, err := cl2.Master()
					if err != nil {
						print(err.Error())
						masterState.WithLabelValues(c.MasterHost, "unknown", "down", "error").Set(0.0)
					} else {
						GeneratePuppetMasterMetrics(cl2, c.MasterHost, false)

					}

				}

				time.Sleep(i * time.Second)
			}
		}()

	} else {
		if c.Debug {
			log.Print("SSL was not configured continuing with http.")

		}
		go func() {
			for {
				cl := puppetdb.NewClient(c.Host, c.Port, false)
				GenerateFactsMetrics(c.Facts, cl, c.Nodes, c.Debug)
				GenerateReportsMetrics(cl, c.Nodes, c.Debug, c.Timeout)
				i := time.Duration(15)
				if c.Interval != 0 {
					i = time.Duration(c.Interval)
				}
				if c.MasterEnable {
					cl2 := puppetdb.NewClientSSLInsecureMaster(c.MasterHost, c.MasterPort, c.Debug)
					_, err := cl2.Master()
					if err != nil {
						print(err.Error())
						masterState.WithLabelValues(c.MasterHost, "unknown", "down", "error").Set(0.0)
					} else {
						GeneratePuppetMasterMetrics(cl2, c.MasterHost, false)

					}
				}

				time.Sleep(i * time.Second)
			}
		}()

	}
	i := 15
	if c.Interval != 0 {
		i = c.Interval
	}
	if c.Debug {
		log.Printf("Starting server on port %d on endpoint %s. Scrape interval is %ds", c.Port, endP, i)
	}
	// Expose the registered metrics via HTTP.
	http.Handle(*endP, promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
			<head><title>Puppetdb Exporter</title></head>
			<body>
			<h1>Puppetdb Exporter</h1>
			<p><a href="` + *endP + `">Metrics</a></p>
			</body>
			</html>`))
	})
	log.Fatal(http.ListenAndServe(*addr, nil))
}
