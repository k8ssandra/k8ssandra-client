package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/k8ssandra/k8ssandra-client/internal/envtest"
	"github.com/stretchr/testify/require"
)

var existingConfig = `
{
	"jvm-server-options": {
	  "initial_heap_size": "512m",
	  "max_heap_size": "512m",
	  "additional-jvm-opts": [
		"-Dcassandra.system_distributed_replication=test-dc:1",
		"-Dcom.sun.management.jmxremote.authenticate=true"
	  ]
	},
	"cassandra-yaml": {
	  "authenticator": "PasswordAuthenticator",
	  "authorizer": "CassandraAuthorizer",
	  "num_tokens": 256,
	  "role_manager": "CassandraRoleManager",
	  "start_rpc": false
	},
	"cluster-info": {
	  "name": "test",
	  "seeds": "test-seed-service,test-dc-additional-seed-service"
	},
	"datacenter-info": {
	  "graph-enabled": 0,
	  "name": "dc1",
	  "solr-enabled": 0,
	  "spark-enabled": 0
	}
}
`

func TestConfigInfoParsing(t *testing.T) {
	require := require.New(t)
	t.Setenv("CONFIG_FILE_DATA", existingConfig)
	configInput, err := parseConfigInput()
	require.NoError(err)
	require.NotNil(configInput)
	require.NotNil(configInput.CassYaml)
	require.NotNil(configInput.ClusterInfo)
	require.NotNil(configInput.DatacenterInfo)

	require.Equal("test", configInput.ClusterInfo.Name)
	require.Equal("dc1", configInput.DatacenterInfo.Name)
}

func TestParseNodeInfo(t *testing.T) {
	require := require.New(t)
	t.Setenv("POD_IP", "172.27.0.1")
	t.Setenv("RACK_NAME", "r1")
	nodeInfo, err := parseNodeInfo()
	require.NoError(err)
	require.NotNil(nodeInfo)
	require.Equal("172.27.0.1", nodeInfo.IP.String())
	require.Equal("r1", nodeInfo.Rack)

	t.Setenv("HOST_IP", "10.0.0.1")
	nodeInfo, err = parseNodeInfo()
	require.NoError(err)
	require.NotNil(nodeInfo)
	require.Equal("172.27.0.1", nodeInfo.IP.String())

	t.Setenv("USE_HOST_IP_FOR_BROADCAST", "false")
	nodeInfo, err = parseNodeInfo()
	require.NoError(err)
	require.NotNil(nodeInfo)
	require.Equal("172.27.0.1", nodeInfo.IP.String())

	t.Setenv("USE_HOST_IP_FOR_BROADCAST", "true")
	nodeInfo, err = parseNodeInfo()
	require.NoError(err)
	require.NotNil(nodeInfo)
	require.Equal("10.0.0.1", nodeInfo.IP.String())
}

func TestCassandraYamlWriting(t *testing.T) {
	require := require.New(t)
	cassYamlDir := filepath.Join(envtest.RootDir(), "testfiles")
	tempDir, err := os.MkdirTemp("", "client-test")

	fmt.Printf("tempDir: %s\n", tempDir)
	require.NoError(err)

	// Create mandatory configs..
	t.Setenv("CONFIG_FILE_DATA", existingConfig)
	configInput, err := parseConfigInput()
	require.NoError(err)
	require.NotNil(configInput)
	t.Setenv("POD_IP", "172.27.0.1")
	t.Setenv("RACK_NAME", "r1")
	nodeInfo, err := parseNodeInfo()
	require.NoError(err)
	require.NotNil(nodeInfo)

	require.NoError(createCassandraYaml(configInput, nodeInfo, cassYamlDir, tempDir))

	// TODO Read back and verify all our changes are there
}

func TestRackProperties(t *testing.T) {
	require := require.New(t)
	propertiesDir := filepath.Join(envtest.RootDir(), "testfiles")
	tempDir, err := os.MkdirTemp("", "client-test")

	fmt.Printf("tempDir: %s\n", tempDir)
	require.NoError(err)

	// Create mandatory configs..
	t.Setenv("CONFIG_FILE_DATA", existingConfig)
	configInput, err := parseConfigInput()
	require.NoError(err)
	require.NotNil(configInput)
	t.Setenv("POD_IP", "172.27.0.1")
	t.Setenv("RACK_NAME", "r1")
	nodeInfo, err := parseNodeInfo()
	require.NoError(err)
	require.NotNil(nodeInfo)

	require.NoError(createRackProperties(configInput, nodeInfo, propertiesDir, tempDir))

	// TODO Verify file data..
}

func TestServerOptionsReading(t *testing.T) {
	require := require.New(t)
	propertiesDir := filepath.Join(envtest.RootDir(), "testfiles")
	inputFile := filepath.Join(propertiesDir, "jvm-server.options")
	s, err := readJvmServerOptions(inputFile)
	require.NoError(err)

	for _, v := range s {
		fmt.Printf("%s\n", v)
	}
}

func TestServerOptionsOutput(t *testing.T) {
	require := require.New(t)
	optionsDir := filepath.Join(envtest.RootDir(), "testfiles")
	tempDir, err := os.MkdirTemp("", "client-test")

	fmt.Printf("tempDir: %s\n", tempDir)
	require.NoError(err)

	// Create mandatory configs..
	t.Setenv("CONFIG_FILE_DATA", existingConfig)
	configInput, err := parseConfigInput()
	require.NoError(err)
	require.NotNil(configInput)

	require.NoError(createJVMOptions(configInput, optionsDir, tempDir))
}
