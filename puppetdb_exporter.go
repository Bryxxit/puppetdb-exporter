package main

import (
	"flag"
	"fmt"
	"github.com/negast/go-puppetdb"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"
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

func addGaugeMetricStatusString(reports []puppetdb.ReportJSON, nodes bool) {
	for _, report := range reports {
		status := "unchanged"
		value := "1"
		if report.Metrics.Data[3].Value > 0 {
			//println("3 " + report.Metrics.Data[3].Name)
			statusTotal.WithLabelValues(report.Metrics.Data[3].Name, report.Environment).Inc()
			statusTotal.WithLabelValues(report.Metrics.Data[3].Name, "All").Inc()
			status = report.Metrics.Data[3].Name
			value = strconv.Itoa(int(report.Metrics.Data[3].Value))
		} else if report.Metrics.Data[2].Value > 0 {
			//println("2 " + report.Metrics.Data[2].Name)
			statusTotal.WithLabelValues(report.Metrics.Data[2].Name, report.Environment).Inc()
			statusTotal.WithLabelValues(report.Metrics.Data[2].Name, "All").Inc()
			status = report.Metrics.Data[2].Name
			value = strconv.Itoa(int(report.Metrics.Data[2].Value))
		} else if report.Metrics.Data[1].Value > 0 {
			//println("1 " + report.Metrics.Data[1].Name)
			statusTotal.WithLabelValues(report.Metrics.Data[1].Name, report.Environment).Inc()
			statusTotal.WithLabelValues(report.Metrics.Data[1].Name, "All").Inc()
			status = report.Metrics.Data[1].Name
			value = strconv.Itoa(int(report.Metrics.Data[1].Value))
		} else {
			//println("0 unchanged")
			statusTotal.WithLabelValues("unchanged", "All").Inc()
			statusTotal.WithLabelValues("unchanged", report.Environment).Inc()
		}

		if nodes {
			statusNodesGuage.WithLabelValues(status, value, report.Environment, report.CertName).Inc()
		}

		// now for the new metrics this might create some duplicates
		for _, metric := range report.Metrics.Data {
			if metric.Category == "time" {
				timeNodeGuage.WithLabelValues(metric.Name, report.Environment, report.CertName).Set(metric.Value)
			} else if metric.Category == "events" {
				eventNodeGuage.WithLabelValues(metric.Name, report.Environment, report.CertName).Set(metric.Value)
			} else if metric.Category == "resources" {
				resourcesNodeGuage.WithLabelValues(metric.Name, report.Environment, report.CertName).Set(metric.Value)
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
	Facts    []string `yaml:"facts"`
	Nodes    bool     `yaml:"nodes"`
	Host     string   `yaml:"host"`
	Port     int      `yaml:"port"`
	SSL      bool     `yaml:"ssl"`
	Key      string   `yaml:"key"`
	Ca       string   `yaml:"ca"`
	Cert     string   `yaml:"cert"`
	Interval int      `yaml:"interval"`
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

func GenerateFactsMetrics(facts []string, c *puppetdb.Client, nodes bool) {
	factTotal.Reset()
	factNodeGuage.Reset()
	for _, fact := range facts {
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

func GenerateReportsMetrics(c *puppetdb.Client, nodes bool) {
	nodesArr, err := c.Nodes()
	if err != nil {
		log.Print(err)
	}
	// Reset nodes values
	statusTotal.Reset()
	statusNodesGuage.Reset()
	resourcesNodeGuage.Reset()
	timeNodeGuage.Reset()
	eventNodeGuage.Reset()

	for _, node := range nodesArr {
		res, err := c.ReportByHash(node.LatestReportHash)
		addGaugeMetricStatusString(res, nodes)
		if err != nil {
			log.Print(err)
		}

	}

}

func main() {
	flag.Parse()

	var c Conf
	c.getConf(*conf)
	if c.Host == "" {
		c.Host = "localhost"
	}
	if c.Port == 0 {
		c.Port = 8080
	}

	if c.SSL {
		go func() {
			for {
				cl := puppetdb.NewClientSSL(c.Host, c.Port, c.Key, c.Cert, c.Ca, false)
				GenerateFactsMetrics(c.Facts, cl, c.Nodes)
				GenerateReportsMetrics(cl, c.Nodes)
				i := time.Duration(15)
				if c.Interval != 0 {
					i = time.Duration(c.Interval)
				}
				time.Sleep(i * time.Second)
			}
		}()

	} else {
		go func() {
			for {
				cl := puppetdb.NewClient(c.Host, c.Port, false)
				GenerateFactsMetrics(c.Facts, cl, c.Nodes)
				GenerateReportsMetrics(cl, c.Nodes)
				i := time.Duration(15)
				if c.Interval != 0 {
					i = time.Duration(c.Interval)
				}
				time.Sleep(i * time.Second)
			}
		}()

	}

	//// Expose the registered metrics via HTTP.
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
