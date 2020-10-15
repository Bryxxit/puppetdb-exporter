# Prometheus PuppetDB exporter

This is our implementation of a prometheus exporter for your Puppet environment. You can gather facts with this and export them to prometheus.
Other things are collected as well such as:
 - Master performance metrics: jruby, jvm, http,...
 - the reports api is also evaluated so that each compile time is exported.
 -
Gathering facts can be done with wildcards. You can now also define fact bundles so that a bunch of facts are set on a single entry.

If you have any requests or issues you can always create an issue on the issues page.

## Quick start

* Clone this repository
```
> git clone https://github.com/Bryxxit/puppetdb-exporter.git
> cd puppetdb-exporter
> go build
```
Alternatively you can download the latest release here
https://github.com/Bryxxit/puppetdb-exporter/releases
The service runs on port 8156 by default so: http://localhost:8156/


* run `./puppetdb_exporter` to start the exporter service

# Puppet exporter
## Command line arguments
```
Usage: puppetdb_exporter [options]
    -listen-address    This is the address and port the listener runs on
    -conf              The path to the configuration file  for the exporter. Default: puppetdb-facts.yaml
    -endpoint          The endpoint where the metrics will be visible in prometheus. Default: /metrics
```

## The config file
The config file is fairly simple and is yaml based. Bellow an example configuration file:
```yaml
---
facts:
  - 'uptime_seconds'
  - 'os.family'
  - 'operatingsystemrelease'
  - 'networking.*'
fact_bundles:
  ips: ["ipaddress_eth2", "ipaddress_eth0", "ipaddress_eth1"]
nodes: true
debug: true
masterEnable: true
host: 'localhost'
port: 8080

``` 

The following table will outline all possible parameters you can use in the configuration.

|parameter      |type           |description|default      |
|---------------|---------------|-----------|-------------|
|`facts`        | Array[String] |an array of facts you want to evaluate. You can use dot notation for subfacts etc. **note** - facts must be a string or number |             |
|`nodes`        | Boolean |Value that determines of you want to display each nodes value for the string based facts |             |
|`host`         | String |Hostname for the PuppetDB | `localhost` |
|`port`         | Number |Port on which PuppetDB is listening | `8080`      |
|`ssl`          | Boolean |Whether you want to use SSL for the PuppetDB connection | `false`     |
|`key`          | String |Path to the key file **note** - mandatory when enabling SSL |             |
|`ca`           | String |Path to the ca file  **note** - mandatory when enabling SSL |             |
|`cert`         | String |Path to the cert file  **note** - mandatory when enabling SSL |             |
|`interval`     | Number |Interval (in seconds) between collections | `15`        |
|`debug`        | Boolean |Determines if you want a log from the exporter or not | `false`     |
|`timeout`      | Number |Time (in seconds) after which the exporter will label a certain node as unresponsive | `3600`      |
|`masterEnable` | Boolean |Whether to gather puppet master metrics or not |             |
|`masterHost`   | String |The puppet master ip/hostname | host        |
|`MasterPort`   | Number |The puppet masters port | `8140`      |
|`fact_bundles`  | map[string][]string | This is a hash where each key holds a list of facts. These will be combined into one label. So you can get al info in one join. FLoat/int is converted to string, wildcards are not allowed


## examples of metrics
### master metrics
puppet_master_jvm_guage{instance="node",job="puppetdb",master="puppet",metric="cpu_usage"}	0.099980004
puppet_master_jvm_guage{instance="node",job="puppetdb",master="puppet",metric="gc_cpu_usage"}	0
puppet_master_jvm_guage{instance="node",job="puppetdb",master="puppet",metric="start_time_ms"}	1598422004112
puppet_master_jvm_guage{instance="node",job="puppetdb",master="puppet",metric="up_time_ms"}	4339825280
....
### reports metrics
puppetdb_report_time_node_guage{name="exec",node="node",puppet_environment="production"} 5.094761
puppetdb_report_time_node_guage{name="fact_generation",node="node",puppet_environment="production"} 1.1460241749882698
....
### facts metrics
puppetdb_facts_bundle_ips_gauge{bundlename="ips",ipaddress_eth0="xxx",ipaddress_eth1="xxx",ipaddress_eth2="xxx",node="node",puppet_environment="development"} 1
puppetdb_facts_total{fact="is_virtual",instance="node",job="puppetdb",value="true"}	xx
...