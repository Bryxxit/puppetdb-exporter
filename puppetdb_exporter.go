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

//  START MODELS AND VARIABLES
var (
	addr = flag.String("listen-address", ":8156", "The address to listen on for HTTP requests.")
	conf = flag.String("conf", "puppetdb-facts.yaml", "The path to the config file.")
	endP = flag.String("endpoint", "/metrics", "The path to display the metrics on default  /metrics")
)

// Conf the configuration file object for this exporter
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

// FactGuageEntry Contains the information for one fact entry
type FactGuageEntry struct {
	Name        string
	Environment string
	CertName    string
	Set         float64
}

// FactNodeGuageEntry Contains the information for one node fact entry
type FactNodeGuageEntry struct {
	Name        string
	Environment string
	Value       string
	CertName    string
	Set         float64
}

// ReportsMetricEntry Holds data for an entry in the reports metric api
type ReportsMetricEntry struct {
	Name        string
	Environment string
	Certname    string
	Value       float64
}

// NodeStatusEntry Holds data for a nodes status
type NodeStatusEntry struct {
	Status      string
	Value       string
	Environment string
	Certname    string
}

//  END MODELS AND VARIABLES

//  START GUAGES
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

var masterHTTPGuage = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "puppet_master_http_guage",
		Help: "Http stats for the master",
	},
	[]string{"master", "route", "type"},
)

var masterHTTPClientGuage = prometheus.NewGaugeVec(
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

// END GUAGES

// START FACT GATHERING METHODS

// GenerateFactsMetrics Scrapes the facts metrics defined in the yaml file and exports them to the prometheus exporter
func GenerateFactsMetrics(facts []string, c *puppetdb.Client, nodes bool, debug bool) {
	if debug {
		log.Println("Resetting facts interfaces.")
	}
	//t1 := time.Now()
	if debug {
		log.Println("Done resetting facts interfaces.")
	}
	// collect metrics
	factArr := map[string]map[string]int{}
	arrG := [][]FactGuageEntry{}
	arrG2 := [][]FactNodeGuageEntry{}
	for _, fact := range facts {
		fA := strings.Split(fact, ".")
		if len(fA) == 1 {
			facts2, _ := getBaseMetric(fA[0], c)
			gatherMetrics(facts2, "", nodes, &factArr, &arrG, &arrG2)

		} else if len(fA) > 1 {
			facts2, _ := getBaseMetric(fA[0], c)
			pathA := append(fA[:0], fA[0+1:]...)
			path := strings.Join(pathA, ".")
			gatherMetrics(facts2, path, nodes, &factArr, &arrG, &arrG2)

		}
	}

	setFactMetrics(factArr, arrG, arrG2, nodes)

}

// gatherMetrics Scrapes the apis and returns the values
func gatherMetrics(facts []puppetdb.FactJSON, path string, nodes bool, factArr *map[string]map[string]int, arrG *[][]FactGuageEntry, arrG2 *[][]FactNodeGuageEntry) {
	arr := []FactGuageEntry{}
	arr2 := []FactNodeGuageEntry{}
	var value interface{}
	if len(facts) > 0 {
		if path != "" {
			value = facts[0].Value.Path(path).Data()
		} else {
			value = facts[0].Value.Data()
		}
		check := reflect.TypeOf(value).String()

		switch check {
		case "int":
			addGaugeMetricfactsIntOrFloatPath2(facts, path, &arr)
			*arrG = append(*arrG, arr)
		case "string":
			addGaugeMetricfactsStringPath2(facts, path, nodes, &arr2, factArr)
			*arrG2 = append(*arrG2, arr2)
		case "float64":
			addGaugeMetricfactsIntOrFloatPath2(facts, path, &arr)
			*arrG = append(*arrG, arr)
		default:
			fmt.Println("Skipping fact because not of type int, float or string")
		}
	}

}

// getBaseMetric Gets the fact for each of the given metric path(assuming correct path is given)
func getBaseMetric(fact string, c *puppetdb.Client) ([]puppetdb.FactJSON, error) {
	facts, err := c.FactPerNode(fact)
	if err != nil {
		log.Println(err)
		puppetDBGuage.Set(0)
	} else {
		puppetDBGuage.Set(1)
	}
	return facts, err
}

// addGaugeMetricfactsIntOrFloatPath2 If given metric was a float this will add a FactGaugeEntry to the array
func addGaugeMetricfactsIntOrFloatPath2(facts []puppetdb.FactJSON, path string, arr *[]FactGuageEntry) {

	for _, fact := range facts {
		var value interface{}
		if path != "" {
			value = fact.Value.Path(path).Data()
			*arr = append(*arr, FactGuageEntry{fact.Name + "." + path, fact.Environment, fact.CertName, value.(float64)})
		} else {
			value = fact.Value.Data()
			*arr = append(*arr, FactGuageEntry{fact.Name, fact.Environment, fact.CertName, value.(float64)})

		}

	}
}

// addGaugeMetricfactsStringPath2 If given metric was a float this will add a FactNodeGaugeEntry to the array and a total fact entry
func addGaugeMetricfactsStringPath2(facts []puppetdb.FactJSON, path string, nodes bool, arr *[]FactNodeGuageEntry, totalArr *map[string]map[string]int) {
	for _, fact := range facts {
		var value string
		if path != "" {
			value = fact.Value.Path(path).Data().(string)
			if _, ok := (*totalArr)[fact.Name+"."+path]; ok {
				if _, ok := (*totalArr)[fact.Name+"."+path][value]; ok {
					(*totalArr)[fact.Name+"."+path][value] = (*totalArr)[fact.Name+"."+path][value] + 1
				} else {
					(*totalArr)[fact.Name+"."+path][value] = 1
				}
			} else {
				(*totalArr)[fact.Name+"."+path] = map[string]int{
					value: 1,
				}
			}
			if nodes {
				*arr = append(*arr, FactNodeGuageEntry{Name: fact.Name, Environment: fact.Environment, Value: value, CertName: fact.CertName, Set: 1})
			}
		} else {
			value = fact.Value.Data().(string)
			if _, ok := (*totalArr)[fact.Name]; ok {
				if _, ok := (*totalArr)[fact.Name][value]; ok {
					(*totalArr)[fact.Name][value] = (*totalArr)[fact.Name][value] + 1
				} else {
					(*totalArr)[fact.Name][value] = 1
				}
			} else {
				(*totalArr)[fact.Name] = map[string]int{
					value: 1,
				}
			}
			if nodes {
				*arr = append(*arr, FactNodeGuageEntry{Name: fact.Name, Environment: fact.Environment, Value: value, CertName: fact.CertName, Set: 1})
			}

		}

	}
}

// setFactMetrics Resets the gauges and exports the given metrics
func setFactMetrics(factArr map[string]map[string]int, arrG [][]FactGuageEntry, arrG2 [][]FactNodeGuageEntry, nodes bool) {
	factTotal.Reset()
	factGuage.Reset()
	factNodeGuage.Reset()
	for key := range factArr {
		for key2, value2 := range factArr[key] {
			factTotal.WithLabelValues(key, key2).Set(float64(value2))
		}
	}
	for _, arr := range arrG {
		for _, nf := range arr {
			factGuage.WithLabelValues(nf.Name, nf.Environment, nf.CertName).Set(nf.Set)
		}
	}
	if nodes {
		for _, arr := range arrG2 {
			for _, nf := range arr {
				factNodeGuage.WithLabelValues(nf.Name, nf.Environment, nf.CertName, nf.Value).Set(1)
			}
		}
	}

}

// END FACT GATHERING METHODS

// START REPORTS GATHERING METHODS

// GenerateReportsMetrics Gathers puppetdb reports metrics and exports them to the exporter
func GenerateReportsMetrics(c *puppetdb.Client, nodes bool, debug bool, timeout int) {
	if debug {
		log.Println("Retrieving nodes.")
	}
	nodesArr, err := c.Nodes()

	if err != nil {
		log.Println(err)
	}
	if debug {
		i := len(nodesArr)
		log.Printf("Nodes collected found %d nodes", i)
		log.Println("Ressting status and node metrics interfaces.")
	}
	// Reset nodes values

	if debug {
		log.Println("Done resetting the interfaces")

	}

	var statusArr map[string]map[string]int = map[string]map[string]int{}
	var timeArr *[]ReportsMetricEntry = &[]ReportsMetricEntry{}
	var eventArr *[]ReportsMetricEntry = &[]ReportsMetricEntry{}
	var resourceArr *[]ReportsMetricEntry = &[]ReportsMetricEntry{}
	var nodeStatusArr *[]NodeStatusEntry = &[]NodeStatusEntry{}

	t1 := time.Now()
	for _, node := range nodesArr {
		// set the report time
		if node.ReportTimestamp != "" {
			// eval time
			layout := "2006-01-02T15:04:05.000Z"
			t, err := time.Parse(layout, node.ReportTimestamp)
			if err != nil {
				log.Println(err.Error())
			}
			duration := time.Since(t)
			timeout := float64(timeout)
			// if timeout reached we do not parse this node
			if duration.Seconds() > timeout {
				setState(node, nodes, debug, int(timeout), &statusArr, nodeStatusArr)

				// if node is okay we parse it
			} else {
				res, err := c.ReportByHash(node.LatestReportHash)
				if debug {
					log.Printf("Retrieving report for node: %s  of environment: %s latest report hash is: %s latest report status is %s latest report time is %s", node.Certname, node.ReportEnvironment, node.LatestReportHash, node.LatestReportStatus, node.ReportTimestamp)
				}
				addGaugeMetricStatusString(res, nodes, node, statusArr, timeArr, eventArr, resourceArr, nodeStatusArr)
				if err != nil {
					log.Println(err)
				}

			}
		} else {
			setState(node, nodes, debug, int(timeout), &statusArr, nodeStatusArr)
		}

	}
	if debug {
		t2 := time.Now()
		diff := t2.Sub(t1)
		log.Println("Retrieving metrics took: " + diff.String())
		//fmt.Println(diff)
	}

	statusTotal.Reset()
	for key := range statusArr {
		for key2, value2 := range statusArr[key] {
			statusTotal.WithLabelValues(key2, key).Set(float64(value2))
		}
	}

	resourcesNodeGuage.Reset()
	t1 = time.Now()
	for _, entry := range *resourceArr {
		resourcesNodeGuage.WithLabelValues(entry.Name, entry.Environment, entry.Certname).Set(entry.Value)
	}
	if debug {
		t2 := time.Now()
		diff := t2.Sub(t1)
		log.Println("Populating resource metrics took: " + diff.String())
	}

	eventNodeGuage.Reset()
	for _, entry := range *timeArr {
		eventNodeGuage.WithLabelValues(entry.Name, entry.Environment, entry.Certname).Set(entry.Value)

	}
	timeNodeGuage.Reset()
	for _, entry := range *timeArr {
		timeNodeGuage.WithLabelValues(entry.Name, entry.Environment, entry.Certname).Set(entry.Value)
	}

	statusNodesGuage.Reset()
	for _, entry := range *nodeStatusArr {
		statusNodesGuage.WithLabelValues(entry.Status, entry.Value, entry.Environment, entry.Certname).Inc()
	}
}

// addGaugeMetricStatusString Fills in metrics for the reports and fills the status array
func addGaugeMetricStatusString(reports []puppetdb.ReportJSON, nodes bool, node puppetdb.NodeJSON,
	statusArr map[string]map[string]int, timeArr *[]ReportsMetricEntry, eventArr *[]ReportsMetricEntry, resourceArr *[]ReportsMetricEntry,
	nodeStatusArr *[]NodeStatusEntry) {
	for _, report := range reports {
		failed := 0.0
		corChanges := 0.0
		changes := 0.0

		for _, metric := range report.Metrics.Data {
			if metric.Category == "time" {
				*timeArr = append(*timeArr, ReportsMetricEntry{metric.Name, report.Environment, report.CertName, metric.Value})
			} else if metric.Category == "events" {
				//eventNodeGuage.WithLabelValues(metric.Name, report.Environment, report.CertName).Set(metric.Value)
				*eventArr = append(*timeArr, ReportsMetricEntry{metric.Name, report.Environment, report.CertName, metric.Value})
			} else if metric.Category == "resources" {
				//resourcesNodeGuage.WithLabelValues(metric.Name, report.Environment, report.CertName).Set(metric.Value)
				*resourceArr = append(*timeArr, ReportsMetricEntry{metric.Name, report.Environment, report.CertName, metric.Value})

				if metric.Name == "failed" {
					failed = metric.Value
				}
				if metric.Name == "changed" {
					changes = metric.Value
				}
				if metric.Name == "corrective_change" {
					corChanges = metric.Value
				}
			}
		}

		stat := node.LatestReportStatus
		if stat == "changed" {
			if corChanges > 0.0 {
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
				//statusNodesGuage.WithLabelValues(report.Status, "1", report.Environment, report.CertName).Inc()
				*nodeStatusArr = append(*nodeStatusArr, NodeStatusEntry{report.Status, "1", report.Environment, report.CertName})
			}
			if node.LatestReportStatus == "failed" {
				value := strconv.Itoa(int(failed))
				*nodeStatusArr = append(*nodeStatusArr, NodeStatusEntry{report.Status, value, report.Environment, report.CertName})
				//statusNodesGuage.WithLabelValues(report.Status, value, report.Environment, report.CertName).Inc()
			}
			if node.LatestReportStatus == "changed" {
				if corChanges > 0 {
					value := strconv.Itoa(int(corChanges))
					*nodeStatusArr = append(*nodeStatusArr, NodeStatusEntry{"corrective_change", value, report.Environment, report.CertName})
					//statusNodesGuage.WithLabelValues("corrective_change", value, report.Environment, report.CertName).Inc()
				}
				if changes > 0 {
					value := strconv.Itoa(int(changes))
					*nodeStatusArr = append(*nodeStatusArr, NodeStatusEntry{"changed", value, report.Environment, report.CertName})
					//statusNodesGuage.WithLabelValues("changed", value, report.Environment, report.CertName).Inc()
				}
			}
		}
	}
}

func setState(node puppetdb.NodeJSON, nodes bool, debug bool, timeout int, statusArr *map[string]map[string]int, nodeStatusArr *[]NodeStatusEntry) {
	if node.ReportTimestamp != "" {
		addLastReportTimeMetric(node)
		// eval time
		layout := "2006-01-02T15:04:05.000Z"
		t, err := time.Parse(layout, node.ReportTimestamp)
		if err != nil {
			log.Println(err.Error())
		}
		duration := time.Since(t)
		timeout2 := float64(timeout)
		if duration.Seconds() > timeout2 {
			if debug {
				log.Printf("Node %s has reached the timeout and is in unresponsive state latest report was %s", node.Certname, node.ReportTimestamp)
			}
			timeoutString := strconv.Itoa(int(duration.Seconds()))

			if nodes {
				*nodeStatusArr = append(*nodeStatusArr, NodeStatusEntry{"unresponsive", timeoutString, node.ReportEnvironment, node.Certname})
				//statusNodesGuage.WithLabelValues("unresponsive", timeoutString, node.ReportEnvironment, node.Certname).Inc()
			}
			setStateInarray(node, "unresponsive", statusArr)

		}
	} else {
		if nodes {
			*nodeStatusArr = append(*nodeStatusArr, NodeStatusEntry{"unreported", "", node.ReportEnvironment, node.Certname})
			//statusNodesGuage.WithLabelValues("unreported", "", node.ReportEnvironment, node.Certname).Inc()
		}
		setStateInarray(node, "unreported", statusArr)

	}
}

func addLastReportTimeMetric(node puppetdb.NodeJSON) {
	if node.ReportTimestamp != "" {
		layout := "2006-01-02T15:04:05.000Z"
		t, err := time.Parse(layout, node.ReportTimestamp)
		if err != nil {
			log.Println(err.Error())
		}
		duration := time.Since(t).Seconds()
		timeLastReportNodeGuage.WithLabelValues(node.ReportEnvironment, node.Certname).Set(duration)
	}

}

func setStateInarray(node puppetdb.NodeJSON, state string, statusArr *map[string]map[string]int) {
	if _, ok := (*statusArr)["All"]; ok {
		if _, ok := (*statusArr)["All"][state]; ok {
			(*statusArr)["All"][state] = (*statusArr)["All"][state] + 1
		} else {
			(*statusArr)["All"][state] = 1
		}
	} else {
		(*statusArr)["All"] = map[string]int{
			state: 1,
		}
	}
	if _, ok := (*statusArr)[node.ReportEnvironment]; ok {
		if _, ok := (*statusArr)[node.ReportEnvironment][state]; ok {
			(*statusArr)[node.ReportEnvironment][state] = (*statusArr)[node.ReportEnvironment][state] + 1
		} else {
			(*statusArr)[node.ReportEnvironment][state] = 1
		}
	} else {
		(*statusArr)[node.ReportEnvironment] = map[string]int{
			state: 1,
		}
	}
}

// END REPORTS GATHERING METHODS

// START PUPPETMASTER METRICS

// GeneratePuppetMasterMetrics Scrapes and exports the metrics found on the puppet master services endpoint
func GeneratePuppetMasterMetrics(cl2 *puppetdb.ClientMaster, host string, debug bool) {
	m, err := cl2.Master()
	if err != nil {
		log.Println(err.Error())
	} else {
		generateHTTPMetrics(&m, host, debug)
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

func generateHTTPMetrics(master *puppetdb.MasterMetrics, host string, debug bool) {
	for _, metric := range *master.Status.Experimental.HttpMetrics {
		masterHTTPGuage.WithLabelValues(host, metric.RouteId, "aggregate").Set(float64(metric.Aggregate))
		masterHTTPGuage.WithLabelValues(host, metric.RouteId, "count").Set(float64(metric.Count))
		masterHTTPGuage.WithLabelValues(host, metric.RouteId, "mean").Set(float64(metric.Mean))
	}

	for _, metric := range *master.Status.Experimental.HttpClientMetrics {
		masterHTTPClientGuage.WithLabelValues(host, metric.MetricName, "aggregate", strings.Join(*metric.MetricId, ",")).Set(float64(metric.Aggregate))
		masterHTTPClientGuage.WithLabelValues(host, metric.MetricName, "count", strings.Join(*metric.MetricId, ",")).Set(float64(metric.Count))
		masterHTTPClientGuage.WithLabelValues(host, metric.MetricName, "mean", strings.Join(*metric.MetricId, ",")).Set(float64(metric.Mean))

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
	//masterFunctionTable.Reset()
	for _, metric := range *p.Status.Experimental.FunctionMetrics {
		masterFunction.WithLabelValues(host, metric.Function, "count").Set(float64(metric.Count))
		masterFunction.WithLabelValues(host, metric.Function, "mean").Set(float64(metric.Mean))
		masterFunction.WithLabelValues(host, metric.Function, "aggregate").Set(float64(metric.Aggregate))
		//masterFunctionTable.WithLabelValues(host, metric.Function, strconv.Itoa(metric.Count),
		//	strconv.Itoa(metric.Mean), strconv.Itoa(metric.Aggregate)).Set(1)
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

// STOP PUPPETMASTER METRICS

// INIT AND MAIN
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

	prometheus.MustRegister(masterHTTPGuage)
	prometheus.MustRegister(masterHTTPClientGuage)

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
	//prometheus.MustRegister(masterFunctionTable)

	prometheus.MustRegister(masterCatalog)
	prometheus.MustRegister(masterPuppetdb)

	prometheus.MustRegister(masterState)

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
			log.Println("SSL was configured continuing with ssl settings on https.")
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
				time.Sleep(i * time.Second)
			}
		}()
		if c.MasterEnable {
			if c.Debug {
				log.Println("Collecting master metrics")
			}
			go func() {
				for {
					i := time.Duration(15)
					if c.Interval != 0 {
						i = time.Duration(c.Interval)
					}
					cl2 := puppetdb.NewClientSSLMaster(c.MasterHost, c.MasterPort, c.Key, c.Cert, c.Ca, c.Debug)
					_, err := cl2.Master()
					if err != nil {
						print(err.Error())
						masterState.WithLabelValues(c.MasterHost, "unknown", "down", "error").Set(0.0)
					} else {
						GeneratePuppetMasterMetrics(cl2, c.MasterHost, false)

					}
					time.Sleep(i * time.Second)
				}
			}()
		}

	} else {
		if c.Debug {
			log.Println("SSL was not configured continuing with http.")

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
				time.Sleep(i * time.Second)
			}
		}()

		if c.MasterEnable {
			if c.Debug {
				log.Println("Collecting master metrics")
			}
			go func() {
				for {
					i := time.Duration(15)
					if c.Interval != 0 {
						i = time.Duration(c.Interval)
					}
					cl2 := puppetdb.NewClientSSLInsecureMaster(c.MasterHost, c.MasterPort, c.Debug)
					_, err := cl2.Master()
					if err != nil {
						print(err.Error())
						masterState.WithLabelValues(c.MasterHost, "unknown", "down", "error").Set(0.0)
					} else {
						GeneratePuppetMasterMetrics(cl2, c.MasterHost, false)

					}
					time.Sleep(i * time.Second)
				}
			}()
		}

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

// END INIT AND MAIN
