package config

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"strconv"

	"github.com/adutra/goalesce"
	"gopkg.in/yaml.v3"
)

/*
  - Need to find the original Cassandra / DSE configuration (the path) in our images
  - Merge given input with what we have there as a default

  - Merge certain keys to different files (only cassandra-yaml -> yamls)
	- Rack information, cluster information etc
	- Was there some other information we might want here?
  - Merge JSON & YAML?
*/

/*
	For NodeInfo struct, these are set by the cass-operator.
	// TODO We did add some more also, add support for them?
	// TODO Also, RACK_NAME and others could be moved to a JSON key which cass-operator could create..
	// TODO Do we need PRODUCT_VERSION for anything anymore?

	{:pod-ip                    (System/getenv "POD_IP")
      :config-file-data          (System/getenv "CONFIG_FILE_DATA")
      :product-version           (System/getenv "PRODUCT_VERSION")
      :rack-name                 (System/getenv "RACK_NAME")
      :product-name              (or (System/getenv "PRODUCT_NAME") "dse")
      :use-host-ip-for-broadcast (or (System/getenv "USE_HOST_IP_FOR_BROADCAST") "false")
      :host-ip                   (System/getenv "HOST_IP")

	// TODO Could we also refactor the POD_IP / HOST_IP processing? Why can't the decision happen in cass-operator?
*/

func Build(ctx context.Context) error {
	// Parse input from cass-operator
	configInput, err := parseConfigInput()
	if err != nil {
		return err
	}

	nodeInfo, err := parseNodeInfo()
	if err != nil {
		return err
	}

	// Create rack information
	if err := createRackProperties(configInput, nodeInfo, outputConfigFileDir()); err != nil {
		return err
	}

	// Create cassandra-env.sh
	if err := createCassandraEnv(configInput, outputConfigFileDir()); err != nil {
		return err
	}

	// Create jvm*-server.options
	if err := createJVMOptions(configInput, outputConfigFileDir()); err != nil {
		return err
	}

	// Create cassandra.yaml
	if err := createCassandraYaml(configInput, nodeInfo, defaultConfigFileDir(), outputConfigFileDir()); err != nil {
		return err
	}

	return nil
}

// Refactor to methods to saner names and files..

func parseConfigInput() (*ConfigInput, error) {
	configInputStr := os.Getenv("CONFIG_FILE_DATA")
	configInput := &ConfigInput{}
	if err := json.Unmarshal([]byte(configInputStr), configInput); err != nil {
		return nil, err
	}

	return configInput, nil
}

func parseNodeInfo() (*NodeInfo, error) {
	rackName := os.Getenv("RACK_NAME")

	n := &NodeInfo{
		Rack: rackName,
	}

	podIp := os.Getenv("POD_IP")

	useHostIp := false
	useHostIpStr := os.Getenv("USE_HOST_IP_FOR_BROADCAST")
	if useHostIpStr != "" {
		var err error
		useHostIp, err = strconv.ParseBool(useHostIpStr)
		if err != nil {
			return nil, err
		}
	}

	if useHostIp {
		podIp = os.Getenv("HOST_IP")
	}

	if ip := net.ParseIP(podIp); ip != nil {
		n.IP = ip
	}

	return n, nil
}

// findConfigFiles returns the path of config files in the cass-management-api (for Cassandra 4.1.x and up)
func defaultConfigFileDir() string {
	// $CASSANDRA_CONF could modify this, but we override it in the mgmt-api
	return "/opt/cassandra/conf"
}

func outputConfigFileDir() string {
	// docker-entrypoint.sh will copy the files from here, so we need all the outputs to target this
	return "/configs"
}

func createRackProperties(configInput *ConfigInput, nodeInfo *NodeInfo, targetDir string) error {
	// Write cassandra-rackdc.properties file with Datacenter and Rack information

	// Load the current file

	// Set dc to DatacenterInfo.Name
	// Set rack to NodeInfo.Rack
	return nil
}

func createCassandraEnv(configInput *ConfigInput, targetDir string) error {
	// Modify cassandra-env.sh if it's in the jvm-options / jvm-server-options / additional-jvm-options?
	// This probably needs a template that is used to ensure backwards compatibility
	return nil
}

func createJVMOptions(configInput *ConfigInput, targetDir string) error {
	return nil
}

// cassandra.yaml related functions

func createCassandraYaml(configInput *ConfigInput, nodeInfo *NodeInfo, sourceDir, targetDir string) error {
	// Read the base file
	yamlPath := filepath.Join(sourceDir, "cassandra.yaml")

	yamlFile, err := os.ReadFile(yamlPath)
	if err != nil {
		return err
	}

	// Unmarshal, Marshal to remove all comments (and some fields if necessary)
	cassandraYaml := make(map[string]interface{})

	if err := yaml.Unmarshal(yamlFile, cassandraYaml); err != nil {
		return err
	}

	// Merge with the ConfigInput's cassadraYaml changes - configInput.CassYaml changes have to take priority
	merged, err := goalesce.DeepMerge(cassandraYaml, configInput.CassYaml)
	if err != nil {
		return err
	}

	// Take the NodeInfo information and add those modifications to the merge output (a priority)
	// Take the mandatory changes we require and merge them (a priority again)
	merged = k8ssandraOverrides(merged, configInput, nodeInfo)

	// Write to the targetDir the new modified file
	targetFile := filepath.Join(targetDir, "cassandra.yaml")
	return writeYaml(merged, targetFile)
}

func k8ssandraOverrides(merged map[string]interface{}, configInput *ConfigInput, nodeInfo *NodeInfo) map[string]interface{} {
	// Add fields which we require and their values, these should override whatever user sets
	merged["seed_provider"] = []map[string]interface{}{
		{
			"class_name": "org.apache.cassandra.locator.K8SeedProvider",
			"parameters": []map[string]interface{}{
				{
					"seeds": configInput.ClusterInfo.Seeds,
				},
			},
		},
	}

	listenIP := nodeInfo.IP.String()
	merged["listen_address"] = listenIP
	merged["rpc_address"] = listenIP
	delete(merged, "broadcast_address")     // Sets it to the same as listen_address
	delete(merged, "rpc_broadcast_address") // Sets it to the same as rpc_address

	return merged
}

func writeYaml(doc map[string]interface{}, targetFile string) error {
	b, err := yaml.Marshal(doc)
	if err != nil {
		return err
	}

	return os.WriteFile(targetFile, b, 0) // TODO Fix Perm
}
