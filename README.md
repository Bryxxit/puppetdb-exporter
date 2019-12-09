# Prometheus PuppetDB exporter

This is our implementation of a prometheus exporter for your Puppet environment. It exports facts from PuppetDB, so you can create metrics about your environment, based on them.

It will also gather information about the status of your Puppet infrastructure (Puppet master health and performance).


## Quick start

* Clone this repository
```
> git clone https://github.com/Bryxxit/puppetdb-exporter.git
> cd puppetdb-exporter
> go build
```
Alternatively you can download the latest release here
https://github.com/Bryxxit/puppetdb-exporter/releases
The servcie runs on port 8156 by default so: http://localhost:8156/


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
