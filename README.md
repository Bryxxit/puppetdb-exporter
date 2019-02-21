# Prometheus PuppetDB exporter

At the moment it is only a way to export facts. So you can create metrics etc about your environment.


# Install 

To install just unzip the latest release and run by ./puppetdb_exporter

# configuration

There are some command line parameters you can set

- listen-address: this is the address and port the listener runs on
- conf: This is the path to the configuration file for the exporter by default this will be puppetdb-facts.yaml
- endpoint: This is the endpoint where the metrics will be visible by default this will be /metrics


# The config file
The config file is fairly short for now and is yaml based following parameters are configurable:

- facts: This is an array of facts in string that you want to evaluate. You can use dot notation to subfacts etc. Note that facts must be a string or number.
    - example "os.family", "uptime_seconds"
- nodes: This is a boolean whether to display each nodes value for the string based facts.
- host: The hostname for the puppetdb default is localhost
- port: The port of the puppetdb default is 8080
- ssl: Whether to use ssl for the connection or not default is disabled.
- key: The path to the key file for ssl
- ca: The path to the ca file for ssl
- cert: The path to the cert file for ssl
- interval: The interval between collections default to 15 seconds