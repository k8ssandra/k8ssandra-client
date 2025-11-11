package config

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/k8ssandra/k8ssandra-client/internal/envtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

var existingConfig = `
{
	"cassandra-env-sh": {
		"malloc-arena-max": 8,
		"additional-jvm-opts": [
		"-Dcassandra.system_distributed_replication=test-dc:1",
		"-Dcom.sun.management.jmxremote.authenticate=true"
		]
	},
	"jvm-server-options": {
		"initial_heap_size": "512m",
		"max_heap_size": "512m",
		"per_thread_stack_size": "384k",
        "java_net_prefer_ipv4_stack": "false",
		"additional-jvm-opts": [
		"-Dcassandra.system_distributed_replication=test-dc:1",
		"-Dcom.sun.management.jmxremote.authenticate=true"
		]
	},
	"jvm11-server-options": {
		"g1r_set_updating_pause_time_percent": "6",
		"additional-jvm-opts": [
		"-XX:MaxGCPauseMillis=350"
		]
	},
	"cassandra-yaml": {
		"authenticator": "PasswordAuthenticator",
		"authorizer": "CassandraAuthorizer",
		"num_tokens": 256,
		"role_manager": "CassandraRoleManager",
		"start_rpc": false,
        "rpc_interface_prefer_ipv6": true
	},
	"cluster-info": {
		"name": "test",
		"seeds": "test-seed-service,test-dc-additional-seed-service"
	},
	"datacenter-info": {
		"graph-enabled": 0,
		"name": "datacenter1",
		"solr-enabled": 0,
		"spark-enabled": 0
	}
}
`

var cass50Config = `
{
	"cassandra-yaml": {
		"authenticator": {
			"class_name": "org.apache.cassandra.auth.PasswordAuthenticator"
		},
		"authorizer": "org.apache.cassandra.auth.CassandraAuthorizer",
		"role_manager": "CassandraRoleManager"
	},
	"cluster-info": {
		"name": "test",
		"seeds": "test-seed-service,test-dc-additional-seed-service"
	},
	"datacenter-info": {
		"graph-enabled": 0,
		"name": "datacenter1",
		"solr-enabled": 0,
		"spark-enabled": 0
	}
}
`

var cass50ConfigJDK17OverrideGC = `
{
	"cassandra-yaml": {
		"authenticator": {
			"class_name": "org.apache.cassandra.auth.PasswordAuthenticator"
		},
		"authorizer": "org.apache.cassandra.auth.CassandraAuthorizer",
		"role_manager": "CassandraRoleManager"
	},
	"jvm17-server-options": {
		"garbage_collector": "Shenandoah",
		"additional-jvm-opts": [
		"-XX:MaxGCPauseMillis=350"
		]
	},
	"cluster-info": {
		"name": "test",
		"seeds": "test-seed-service,test-dc-additional-seed-service"
	},
	"datacenter-info": {
		"graph-enabled": 0,
		"name": "datacenter1",
		"solr-enabled": 0,
		"spark-enabled": 0
	}
}
`

var cass50ConfigJDK17OverrideGCFromAdditionalOpts = `
{
	"cassandra-yaml": {
		"authenticator": {
			"class_name": "org.apache.cassandra.auth.PasswordAuthenticator"
		},
		"authorizer": "org.apache.cassandra.auth.CassandraAuthorizer",
		"role_manager": "CassandraRoleManager"
	},
	"jvm17-server-options": {
		"additional-jvm-opts": [
		  "-XX:+UseZGC"
		]
	},
	"cluster-info": {
		"name": "test",
		"seeds": "test-seed-service,test-dc-additional-seed-service"
	},
	"datacenter-info": {
		"graph-enabled": 0,
		"name": "datacenter1",
		"solr-enabled": 0,
		"spark-enabled": 0
	}
}
`

var numericConfig = `
{
	"jvm-server-options": {
		"max_heap_size": 524288000
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
		"name": "dc2",
		"solr-enabled": 0,
		"spark-enabled": 0
	}
}
`

var booleanOverride = `
{
    "cassandra-yaml": {
        "authenticator": "com.datastax.bdp.cassandra.auth.DseAuthenticator",
        "authorizer": "com.datastax.bdp.cassandra.auth.DseAuthorizer",
        "auto_snapshot": false,
        "file_cache_size_in_mb": 100,
        "memtable_space_in_mb": 100,
        "num_tokens": 0,
        "role_manager": "com.datastax.bdp.cassandra.auth.DseRoleManager",
        "rpc_keepalive": false,
        "rpc_address": "1.1.1.1"
    },
    "cluster-info": {
        "name": "cluster1",
        "seeds": "cluster1-seed-service,cluster1-dc1-additional-seed-service"
    },
    "datacenter-info": {
        "graph-enabled": 0,
        "name": "dc1",
        "solr-enabled": 0,
        "spark-enabled": 1
    },
    "jvm-server-options": {
        "initial_heap_size": "2000m",
        "max_heap_size": "2000m"
    }
}`

func TestBuilderDefaults(t *testing.T) {
	require := require.New(t)
	builder := NewBuilder("", "")
	require.Equal(defaultInputDir, builder.configInputDir)
	require.Equal(defaultOutputDir, builder.configOutputDir)
}

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
	require.Equal("datacenter1", configInput.DatacenterInfo.Name)
}

func TestParseNodeInfo(t *testing.T) {
	require := require.New(t)
	t.Setenv("POD_IP", "172.27.0.1")
	t.Setenv("RACK_NAME", "r1")
	nodeInfo, err := parseNodeInfo()
	require.NoError(err)
	require.NotNil(nodeInfo)
	require.Equal("172.27.0.1", nodeInfo.ListenIP.String())
	require.Equal("172.27.0.1", nodeInfo.BroadcastIP.String())
	require.Equal("0.0.0.0", nodeInfo.RPCIP.String())
	require.Equal("r1", nodeInfo.Rack)

	t.Setenv("POD_IP", "fd00:10:244:4::7")
	nodeInfo, err = parseNodeInfo()
	require.NoError(err)
	require.NotNil(nodeInfo)
	require.Equal("fd00:10:244:4::7", nodeInfo.ListenIP.String())
	require.Equal("fd00:10:244:4::7", nodeInfo.BroadcastIP.String())
	require.Equal("::", nodeInfo.RPCIP.String())
	require.Equal("r1", nodeInfo.Rack)

	t.Setenv("POD_IP", "172.27.0.1")
	t.Setenv("HOST_IP", "10.0.0.1")
	nodeInfo, err = parseNodeInfo()
	require.NoError(err)
	require.NotNil(nodeInfo)
	require.Equal("172.27.0.1", nodeInfo.ListenIP.String())
	require.Equal("172.27.0.1", nodeInfo.BroadcastIP.String())

	t.Setenv("USE_HOST_IP_FOR_BROADCAST", "false")
	nodeInfo, err = parseNodeInfo()
	require.NoError(err)
	require.NotNil(nodeInfo)
	require.Equal("172.27.0.1", nodeInfo.ListenIP.String())
	require.Equal("172.27.0.1", nodeInfo.BroadcastIP.String())

	t.Setenv("USE_HOST_IP_FOR_BROADCAST", "true")
	nodeInfo, err = parseNodeInfo()
	require.NoError(err)
	require.NotNil(nodeInfo)
	require.Equal("172.27.0.1", nodeInfo.ListenIP.String())
	require.Equal("10.0.0.1", nodeInfo.BroadcastIP.String())
}

func TestBuild(t *testing.T) {
	require := require.New(t)
	t.Setenv("CONFIG_FILE_DATA", existingConfig)
	inputDir := filepath.Join(envtest.RootDir(), "testfiles")
	tempDir, err := os.MkdirTemp("", "client-test")
	require.NoError(err)
	defer os.RemoveAll(tempDir)

	b := NewBuilder(inputDir, tempDir)
	require.NoError(b.Build(context.TODO()))

	// Verify that all target files are there..
	entries, err := os.ReadDir(tempDir)
	require.NoError(err)

	fileNames := make([]string, 0, len(entries))
	for _, v := range entries {
		fileNames = append(fileNames, v.Name())
		f, err := v.Info()
		require.NoError(err)
		require.True(f.Size() > 0)
	}

	require.Contains(fileNames, "cassandra-env.sh")
	require.Contains(fileNames, "cassandra-rackdc.properties")
	require.Contains(fileNames, "cassandra.yaml")
	require.Contains(fileNames, "jvm-server.options")
	require.Contains(fileNames, "jvm11-server.options")
}

func TestCassandraYamlWriting(t *testing.T) {
	require := require.New(t)
	cassYamlDir := filepath.Join(envtest.RootDir(), "testfiles")
	tempDir, err := os.MkdirTemp("", "client-test")
	require.NoError(err)

	defer os.RemoveAll(tempDir)

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

	yamlOrigPath := filepath.Join(cassYamlDir, "cassandra_latest.yaml")
	yamlPath := filepath.Join(tempDir, "cassandra.yaml")

	yamlOrigFile, err := os.ReadFile(yamlOrigPath)
	require.NoError(err)

	yamlFile, err := os.ReadFile(yamlPath)
	require.NoError(err)

	// Unmarshal, Marshal to remove all comments (and some fields if necessary)
	cassandraYaml := make(map[string]interface{})
	require.NoError(yaml.Unmarshal(yamlFile, cassandraYaml))

	cassandraOrigYaml := make(map[string]interface{})
	require.NoError(yaml.Unmarshal(yamlOrigFile, cassandraOrigYaml))

	// Verify all the original keys are there (nothing was removed)
	for k := range cassandraOrigYaml {
		require.Contains(cassandraYaml, k)
	}

	// Verify our k8ssandra overrides are set
	clusterName := configInput.ClusterInfo.Name
	require.Equal(clusterName, cassandraYaml["cluster_name"])

	seedProviders := cassandraYaml["seed_provider"].([]interface{})
	seedProvider := seedProviders[0].(map[string]interface{})
	require.Equal("org.apache.cassandra.locator.K8SeedProvider", seedProvider["class_name"])
	require.Equal("GossipingPropertyFileSnitch", cassandraYaml["endpoint_snitch"])

	listenIP := nodeInfo.ListenIP.String()
	require.Equal(listenIP, cassandraYaml["listen_address"])

	// Verify our changed properties are there
	require.Equal("PasswordAuthenticator", cassandraYaml["authenticator"])
	require.Equal("CassandraAuthorizer", cassandraYaml["authorizer"])
	require.Equal("CassandraRoleManager", cassandraYaml["role_manager"])
	require.Equal("256", cassandraYaml["num_tokens"])
	require.Equal(false, cassandraYaml["start_rpc"])
	require.Equal("0.0.0.0", cassandraYaml["rpc_address"])
}

func TestCassandraBaseConfigFilePick(t *testing.T) {
	require := require.New(t)
	testFilesPath := filepath.Join(envtest.RootDir(), "testfiles")

	// Create input directories and copy correct files to them
	inputDirOld, err := os.MkdirTemp("", "cassandra-yaml")
	require.NoError(err)

	inputDirNew, err := os.MkdirTemp("", "cassandra-yaml")
	require.NoError(err)

	t.Cleanup(func() {
		os.RemoveAll(inputDirOld)
		os.RemoveAll(inputDirNew)
	})

	// Copy the correct files to the directories
	require.NoError(copyFile(filepath.Join(testFilesPath, "cassandra.yaml"), filepath.Join(inputDirOld, "cassandra.yaml")))
	require.NoError(copyFile(filepath.Join(testFilesPath, "cassandra.yaml"), filepath.Join(inputDirNew, "cassandra.yaml")))
	require.NoError(copyFile(filepath.Join(testFilesPath, "cassandra_latest.yaml"), filepath.Join(inputDirNew, "cassandra_latest.yaml")))

	// Then process both..
	outputDirOld, err := os.MkdirTemp("", "cassandra-yaml")
	require.NoError(err)
	outputDirNew, err := os.MkdirTemp("", "cassandra-yaml")
	require.NoError(err)

	t.Cleanup(func() {
		os.RemoveAll(outputDirOld)
		os.RemoveAll(outputDirNew)
	})

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

	require.NoError(createCassandraYaml(configInput, nodeInfo, inputDirOld, outputDirOld))
	require.NoError(createCassandraYaml(configInput, nodeInfo, inputDirNew, outputDirNew))

	// Verify only cassandra.yaml is created to destination
	entriesOld, err := os.ReadDir(outputDirOld)
	require.NoError(err)
	require.Len(entriesOld, 1)
	require.Equal("cassandra.yaml", entriesOld[0].Name())

	entriesNew, err := os.ReadDir(outputDirNew)
	require.NoError(err)
	require.Len(entriesNew, 1)
	require.Equal("cassandra.yaml", entriesNew[0].Name())

	// Verify content differences (that we actually used the _latest when it's present)
	yamlOldOutput := filepath.Join(outputDirOld, "cassandra.yaml")
	yamlNewOutput := filepath.Join(outputDirNew, "cassandra.yaml")

	yamlOldFile, err := os.ReadFile(yamlOldOutput)
	require.NoError(err)

	yamlNewFile, err := os.ReadFile(yamlNewOutput)
	require.NoError(err)

	cassandraYamlOld := make(map[string]interface{})
	require.NoError(yaml.Unmarshal(yamlOldFile, cassandraYamlOld))

	cassandraYamlNew := make(map[string]interface{})
	require.NoError(yaml.Unmarshal(yamlNewFile, cassandraYamlNew))

	require.Equal("heap_buffers", cassandraYamlOld["memtable_allocation_type"])
	require.Equal("offheap_objects", cassandraYamlNew["memtable_allocation_type"])
}

func TestCassandraYamlSubPath(t *testing.T) {
	require := require.New(t)
	cassYamlDir := filepath.Join(envtest.RootDir(), "testfiles")
	tempDir, err := os.MkdirTemp("", "client-test")
	require.NoError(err)

	defer os.RemoveAll(tempDir)

	// Create mandatory configs..
	t.Setenv("CONFIG_FILE_DATA", cass50Config)
	configInput, err := parseConfigInput()
	require.NoError(err)
	require.NotNil(configInput)
	t.Setenv("POD_IP", "172.27.0.1")
	t.Setenv("RACK_NAME", "r1")
	nodeInfo, err := parseNodeInfo()
	require.NoError(err)
	require.NotNil(nodeInfo)

	require.NoError(createCassandraYaml(configInput, nodeInfo, cassYamlDir, tempDir))

	yamlPath := filepath.Join(tempDir, "cassandra.yaml")

	yamlFile, err := os.ReadFile(yamlPath)
	require.NoError(err)

	// Unmarshal, Marshal to remove all comments (and some fields if necessary)
	cassandraYaml := make(map[string]interface{})
	require.NoError(yaml.Unmarshal(yamlFile, cassandraYaml))

	authenticator := cassandraYaml["authenticator"]
	authenticatorStruct := authenticator.(map[string]interface{})
	require.Equal("org.apache.cassandra.auth.PasswordAuthenticator", authenticatorStruct["class_name"])
}

func TestBooleanOverride(t *testing.T) {
	require := require.New(t)
	cassYamlDir := filepath.Join(envtest.RootDir(), "testfiles")
	tempDir, err := os.MkdirTemp("", "client-test")
	require.NoError(err)

	defer os.RemoveAll(tempDir)

	// Create mandatory configs..
	t.Setenv("CONFIG_FILE_DATA", booleanOverride)
	configInput, err := parseConfigInput()
	require.NoError(err)
	require.NotNil(configInput)
	t.Setenv("POD_IP", "172.27.0.1")
	t.Setenv("RACK_NAME", "r1")
	nodeInfo, err := parseNodeInfo()
	require.NoError(err)
	require.NotNil(nodeInfo)

	require.NoError(createCassandraYaml(configInput, nodeInfo, cassYamlDir, tempDir))

	yamlPath := filepath.Join(tempDir, "cassandra.yaml")

	yamlFile, err := os.ReadFile(yamlPath)
	require.NoError(err)

	// Unmarshal, Marshal to remove all comments (and some fields if necessary)
	cassandraYaml := make(map[string]interface{})
	require.NoError(yaml.Unmarshal(yamlFile, cassandraYaml))

	authenticator := cassandraYaml["authenticator"]
	require.Equal("com.datastax.bdp.cassandra.auth.DseAuthenticator", authenticator)
	require.Equal(false, cassandraYaml["auto_snapshot"])
	require.Equal(false, cassandraYaml["rpc_keepalive"])
	require.Equal("1.1.1.1", cassandraYaml["rpc_address"])
}

func TestRackProperties(t *testing.T) {
	require := require.New(t)
	propertiesDir := filepath.Join(envtest.RootDir(), "testfiles")
	tempDir, err := os.MkdirTemp("", "client-test")
	defer os.RemoveAll(tempDir)
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

	lines, err := readFileToLines(tempDir, "cassandra-rackdc.properties")
	require.NoError(err)
	require.Equal(2, len(lines))
	require.Contains(lines, "dc=datacenter1")
	require.Contains(lines, "rack=r1")
}

func TestServerOptionsOutput(t *testing.T) {
	require := require.New(t)
	optionsDir := filepath.Join(envtest.RootDir(), "testfiles")
	tempDir, err := os.MkdirTemp("", "client-test")

	defer os.RemoveAll(tempDir)
	require.NoError(err)

	// Create mandatory configs..
	t.Setenv("CONFIG_FILE_DATA", existingConfig)
	configInput, err := parseConfigInput()
	require.NoError(err)
	require.NotNil(configInput)

	require.NoError(createJVMOptions(configInput, optionsDir, tempDir))

	inputFile := filepath.Join(tempDir, "jvm-server.options")
	inputFile11 := filepath.Join(tempDir, "jvm11-server.options")

	s, err := readJvmServerOptions(inputFile)
	require.NoError(err)

	require.Contains(s, "-Xss384k")
	require.NotContains(s, "-Xss256k")

	require.Contains(s, "-Xmx512m")
	require.Contains(s, "-Xms512m")
	require.Contains(s, "-Dcassandra.system_distributed_replication=test-dc:1")
	require.Contains(s, "-Dcom.sun.management.jmxremote.authenticate=true")

	s11, err := readJvmServerOptions(inputFile11)

	require.NoError(err)

	require.Contains(s11, "-XX:MaxGCPauseMillis=350")
	require.NotContains(s11, "-XX:MaxGCPauseMillis=300")
	require.Contains(s11, "-XX:G1RSetUpdatingPauseTimePercent=6")
	require.NotContains(s11, "-XX:G1RSetUpdatingPauseTimePercent=5")

	for _, v := range defaultG1Settings {
		if v == "-XX:G1RSetUpdatingPauseTimePercent=5" || v == "-XX:MaxGCPauseMillis=300" {
			// Our config replaces these default values with new values, so they should not be here
			require.NotContains(s11, v)
			continue
		}
		require.Contains(s11, v)
	}

	// Test empty also and check we get the default G1 settings
	ci := &ConfigInput{}
	tempDir2, err := os.MkdirTemp("", "client-test")
	require.NoError(err)
	defer os.RemoveAll(tempDir2)
	require.NoError(createJVMOptions(ci, optionsDir, tempDir2))

	inputFile11 = filepath.Join(tempDir2, "jvm11-server.options")

	s11, err = readJvmServerOptions(inputFile11)
	require.NoError(err)

	for _, v := range defaultG1Settings {
		require.Contains(s11, v)
	}

	// Test CMS option also
	ci = &ConfigInput{
		ServerOptions11: map[string]interface{}{
			"garbage_collector": "CMS",
		},
	}

	tempDir3, err := os.MkdirTemp("", "client-test")
	require.NoError(err)
	defer os.RemoveAll(tempDir3)
	require.NoError(createJVMOptions(ci, optionsDir, tempDir3))

	inputFile11 = filepath.Join(tempDir3, "jvm11-server.options")

	s11, err = readJvmServerOptions(inputFile11)
	require.NoError(err)

	for _, v := range defaultCMSSettings {
		require.Contains(s11, v)
	}
}

func TestGCOptions(t *testing.T) {
	assert := assert.New(t)
	assert.Equal(defaultG1Settings, getGCOptions("G1GC", 11))
	assert.Equal([]string{"-XX:+UseG1GC"}, getGCOptions("G1GC", 17))

	assert.Equal(defaultCMSSettings, getGCOptions("CMS", 11))
	assert.Equal([]string{}, getGCOptions("CMS", 17))

	assert.Equal([]string{"-XX:+UseShenandoahGC"}, getGCOptions("Shenandoah", 11))
	assert.Equal([]string{"-XX:+UseShenandoahGC"}, getGCOptions("Shenandoah", 17))

	assert.Equal([]string{"-XX:+UnlockExperimentalVMOptions", "-XX:+UseZGC"}, getGCOptions("ZGC", 11))
	assert.Equal([]string{"-XX:+UseZGC"}, getGCOptions("ZGC", 17))
}

func TestJVM17GarbageCollectorOptions(t *testing.T) {
	require := require.New(t)
	optionsDir := filepath.Join(envtest.RootDir(), "testfiles")

	// Test G1GC for JVM17
	tempDirG1, err := os.MkdirTemp("", "jvm17-g1gc-test")
	require.NoError(err)
	defer os.RemoveAll(tempDirG1)

	ciG1 := &ConfigInput{
		ServerOptions17: map[string]interface{}{
			"garbage_collector": "G1GC",
		},
	}

	require.NoError(createJVMOptions(ciG1, optionsDir, tempDirG1))

	jvm17FileG1 := filepath.Join(tempDirG1, "jvm17-server.options")
	optionsG1, err := readJvmServerOptions(jvm17FileG1)
	require.NoError(err)

	g1gcFound := false
	for _, opt := range optionsG1 {
		if opt == "-XX:+UseG1GC" {
			g1gcFound = true
		}
		require.NotEqual("-XX:+UseZGC", opt)
		require.NotEqual("-XX:+UseShenandoahGC", opt)
	}
	require.True(g1gcFound, "G1GC option should be present")

	// Test ZGC for JVM17
	tempDirZ, err := os.MkdirTemp("", "jvm17-zgc-test")
	require.NoError(err)
	defer os.RemoveAll(tempDirZ)

	ciZ := &ConfigInput{
		ServerOptions17: map[string]interface{}{
			"garbage_collector": "ZGC",
		},
	}

	require.NoError(createJVMOptions(ciZ, optionsDir, tempDirZ))

	jvm17FileZ := filepath.Join(tempDirZ, "jvm17-server.options")
	optionsZ, err := readJvmServerOptions(jvm17FileZ)
	require.NoError(err)

	zgcFound := false
	for _, opt := range optionsZ {
		if opt == "-XX:+UseZGC" {
			zgcFound = true
		}
		require.NotEqual("-XX:+UseG1GC", opt)
		require.NotEqual("-XX:+UseShenandoahGC", opt)
	}
	require.True(zgcFound, "ZGC option should be present")

	// Test Shenandoah for JVM17
	tempDirS, err := os.MkdirTemp("", "jvm17-shenandoah-test")
	require.NoError(err)
	defer os.RemoveAll(tempDirS)

	ciS := &ConfigInput{
		ServerOptions17: map[string]interface{}{
			"garbage_collector": "Shenandoah",
		},
	}

	require.NoError(createJVMOptions(ciS, optionsDir, tempDirS))

	jvm17FileS := filepath.Join(tempDirS, "jvm17-server.options")
	optionsS, err := readJvmServerOptions(jvm17FileS)
	require.NoError(err)

	shenandoahFound := false
	for _, opt := range optionsS {
		if opt == "-XX:+UseShenandoahGC" {
			shenandoahFound = true
		}
		require.NotEqual("-XX:+UseG1GC", opt)
		require.NotEqual("-XX:+UseZGC", opt)
	}
	require.True(shenandoahFound, "Shenandoah option should be present")
}

func TestCassandraEnv(t *testing.T) {
	require := require.New(t)
	envDir := filepath.Join(envtest.RootDir(), "testfiles")
	tempDir, err := os.MkdirTemp("", "client-test")
	defer os.RemoveAll(tempDir)

	require.NoError(err)

	// Create mandatory configs..
	t.Setenv("CONFIG_FILE_DATA", existingConfig)
	configInput, err := parseConfigInput()
	require.NoError(err)
	require.NotNil(configInput)

	require.NoError(createCassandraEnv(configInput, envDir, tempDir))

	// Verify output
	lines, err := readFileToLines(tempDir, "cassandra-env.sh")
	require.NoError(err)

	require.Contains(lines, "export MALLOC_ARENA_MAX=8")
	require.Contains(lines, "JVM_OPTS=\"$JVM_OPTS -Dcassandra.system_distributed_replication=test-dc:1\"")
	require.Contains(lines, "JVM_OPTS=\"$JVM_OPTS -Dcom.sun.management.jmxremote.authenticate=true\"")
}

func TestReadOptionsWithNumeric(t *testing.T) {
	// JSON Unmarshalling does not Unmarshal everything to type string, instead they can be int/floats/bool etc
	require := require.New(t)

	optionsDir := filepath.Join(envtest.RootDir(), "testfiles")
	tempDir, err := os.MkdirTemp("", "client-test")

	defer os.RemoveAll(tempDir)
	require.NoError(err)

	t.Setenv("CONFIG_FILE_DATA", numericConfig)
	configInput, err := parseConfigInput()
	require.NoError(err)
	require.NotNil(configInput)

	require.NoError(createJVMOptions(configInput, optionsDir, tempDir))

	lines, err := readFileToLines(tempDir, "jvm-server.options")
	require.NoError(err)

	require.Contains(lines, "-Xmx524288000")
}

func TestCass50GCOverrides(t *testing.T) {
	require := require.New(t)
	cassYamlDir := filepath.Join(envtest.RootDir(), "testfiles")
	tempDir, err := os.MkdirTemp("", "client-test")
	require.NoError(err)

	defer os.RemoveAll(tempDir)

	// Create mandatory configs..
	t.Setenv("CONFIG_FILE_DATA", cass50ConfigJDK17OverrideGC)
	configInput, err := parseConfigInput()
	require.NoError(err)
	require.NotNil(configInput)
	t.Setenv("POD_IP", "172.27.0.1")
	t.Setenv("RACK_NAME", "r1")
	nodeInfo, err := parseNodeInfo()
	require.NoError(err)
	require.NotNil(nodeInfo)

	require.NoError(createJVMOptions(configInput, cassYamlDir, tempDir))

	jvm17OptionsFile := filepath.Join(tempDir, "jvm17-server.options")
	options, err := readJvmServerOptions(jvm17OptionsFile)
	require.NoError(err)

	require.NotContains(options, "-XX:+UseZGC")
	require.NotContains(options, "-XX:+UseG1GC")
	require.Contains(options, "-XX:+UseShenandoahGC")
}

func TestCass50GCOverridesAdditionalOpts(t *testing.T) {
	require := require.New(t)
	cassYamlDir := filepath.Join(envtest.RootDir(), "testfiles")
	tempDir, err := os.MkdirTemp("", "client-test")
	require.NoError(err)

	defer os.RemoveAll(tempDir)

	// Create mandatory configs..
	t.Setenv("CONFIG_FILE_DATA", cass50ConfigJDK17OverrideGCFromAdditionalOpts)
	configInput, err := parseConfigInput()
	require.NoError(err)
	require.NotNil(configInput)
	t.Setenv("POD_IP", "172.27.0.1")
	t.Setenv("RACK_NAME", "r1")
	nodeInfo, err := parseNodeInfo()
	require.NoError(err)
	require.NotNil(nodeInfo)

	require.NoError(createJVMOptions(configInput, cassYamlDir, tempDir))

	jvm17OptionsFile := filepath.Join(tempDir, "jvm17-server.options")
	options, err := readJvmServerOptions(jvm17OptionsFile)
	require.NoError(err)

	require.Contains(options, "-XX:+UseZGC")
	require.NotContains(options, "-XX:+UseG1GC")
	require.NotContains(options, "-XX:+UseShenandoahGC")
}

// readFileToLines is a small test helper, reads file to []string (per line). This version does not filter anything, not even whitespace.
func readFileToLines(dir, filename string) ([]string, error) {
	outputFile := filepath.Join(dir, filename)
	lines := make([]string, 0, 1)

	f, err := os.Open(outputFile)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}

func TestCopyFiles(t *testing.T) {
	require := require.New(t)
	inputDir := filepath.Join(envtest.RootDir(), "testfiles")
	tempDir, err := os.MkdirTemp("", "client-test")
	require.NoError(err)
	defer os.RemoveAll(tempDir)

	require.NoError(copyFiles(inputDir, tempDir))

	// We should have tempDir/jvm11-clients.options
	_, err = os.Stat(filepath.Join(tempDir, "jvm11-clients.options"))
	require.NoError(err)
}
