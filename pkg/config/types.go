package config

import "net"

/*
	(defrecord ClusterInfo [name seeds])
	(defrecord DatacenterInfo [name
							graph-enabled
							solr-enabled
							spark-enabled])
	;; Note, we are using the current _address field names as of DSE 6.0.
	;; Namely native_transport_address and native_transport_rpc_address.
	;; Clients should not be passing in the old names.
	(defrecord NodeInfo [name
						rack
						listen_address
						broadcast_address
						native_transport_address
						native_transport_broadcast_address
						initial_token
						auto_bootstrap
						agent_version
						configured-paths
						facts])
*/

// From cass-operator JSON input

type ConfigInput struct {
	ClusterInfo    ClusterInfo            `json:"cluster-info"`
	DatacenterInfo DatacenterInfo         `json:"datacenter-info"`
	CassYaml       map[string]interface{} `json:"cassandra-yaml,omitempty"`
	ServerOptions  map[string]interface{} `json:"jvm-server-options,omitempty"`

	// At some point, parse the remaining unknown keys when we decide what to do with them..
}

type ClusterInfo struct {
	Name  string `json:"name"`
	Seeds string `json:"seeds"` // comma separated list of seeds
}

type DatacenterInfo struct {
	Name string `json:"name"`

	// These are ignored for now
	// "graph-enabled": graphEnabled,
	// "solr-enabled":  solrEnabled,
	// "spark-enabled": sparkEnabled,
}

// Built from other sources

type NodeInfo struct {
	Name string
	Rack string
	IP   net.IP
}
