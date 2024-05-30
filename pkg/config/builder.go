package config

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"github.com/adutra/goalesce"
	metadata "github.com/burmanm/definitions-parser/pkg/types"
	gentypes "github.com/burmanm/definitions-parser/pkg/types/generated"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

const (
	// $CASSANDRA_CONF could modify this, but we override it in the mgmt-api
	defaultInputDir = "/cassandra-base-config"

	// docker-entrypoint.sh will copy the files from here, so we need all the outputs to target this
	defaultOutputDir = "/config"

	oldCassandraConfigName    = "cassandra.yaml"
	latestCassandraConfigName = "cassandra_latest.yaml"
)

type Builder struct {
	configInputDir  string
	configOutputDir string
}

func NewBuilder(overrideConfigInput, overrideConfigOutput string) *Builder {
	b := &Builder{
		configInputDir:  defaultInputDir,
		configOutputDir: defaultOutputDir,
	}

	if overrideConfigInput != "" {
		b.configInputDir = overrideConfigInput
	}

	if overrideConfigOutput != "" {
		b.configOutputDir = overrideConfigOutput
	}

	return b
}

var (
	prefixRegexp = regexp.MustCompile(gentypes.JvmServerOptionsPrefixExp)
)

func (b *Builder) Build(ctx context.Context) error {
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
	if err := createRackProperties(configInput, nodeInfo, b.configInputDir, b.configOutputDir); err != nil {
		return err
	}

	// Create cassandra-env.sh
	if err := createCassandraEnv(configInput, b.configInputDir, b.configOutputDir); err != nil {
		return err
	}

	// Create jvm*-server.options
	if err := createJVMOptions(configInput, b.configInputDir, b.configOutputDir); err != nil {
		return err
	}

	// Create cassandra.yaml
	if err := createCassandraYaml(configInput, nodeInfo, b.configInputDir, b.configOutputDir); err != nil {
		return err
	}

	// Copy files which we're not modifying
	if err := copyFiles(b.configInputDir, b.configOutputDir); err != nil {
		return err
	}

	return nil
}

// Refactor to methods to saner names and files..

func parseConfigInput() (*ConfigInput, error) {
	configInputStr := os.Getenv("CONFIG_FILE_DATA")
	configInput := &ConfigInput{}

	d := json.NewDecoder(strings.NewReader(configInputStr))
	d.UseNumber() // This decodes the numbers as strings
	if err := d.Decode(configInput); err != nil {
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
	hostIp := os.Getenv("HOST_IP")

	useHostIp := false
	useHostIpStr := os.Getenv("USE_HOST_IP_FOR_BROADCAST")
	if useHostIpStr != "" {
		var err error
		useHostIp, err = strconv.ParseBool(useHostIpStr)
		if err != nil {
			return nil, err
		}
	}

	broadcastIp := podIp
	if useHostIp {
		broadcastIp = hostIp
	}

	if ip := net.ParseIP(broadcastIp); ip != nil {
		n.BroadcastIP = ip
	}

	if ip := net.ParseIP(podIp); ip != nil {
		n.ListenIP = ip
	}

	// This is not currently overridable
	if ip := net.ParseIP("0.0.0.0"); ip != nil {
		n.RPCIP = ip
	}

	return n, nil
}

func createRackProperties(configInput *ConfigInput, nodeInfo *NodeInfo, sourceDir, targetDir string) error {
	// This creates the cassandra-rackdc.properites with a template with only the values we currently support
	targetFileT := filepath.Join(targetDir, "cassandra-rackdc.properties")
	fT, err := os.OpenFile(targetFileT, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0770)
	if err != nil {
		return err
	}

	defer fT.Close()

	rackTemplate, err := template.New("cassandra-rackdc.properties").Parse("dc={{ .DatacenterName }}\nrack={{ .RackName }}\n")
	if err != nil {
		return err
	}

	type RackTemplate struct {
		DatacenterName string
		RackName       string
	}

	rt := RackTemplate{
		DatacenterName: configInput.DatacenterInfo.Name,
		RackName:       nodeInfo.Rack,
	}

	return rackTemplate.Execute(fT, rt)
}

func createCassandraEnv(configInput *ConfigInput, sourceDir, targetDir string) error {
	envPath := filepath.Join(sourceDir, "cassandra-env.sh")
	f, err := os.ReadFile(envPath)
	if err != nil {
		return err
	}

	targetFileT := filepath.Join(targetDir, "cassandra-env.sh")
	fT, err := os.OpenFile(targetFileT, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0770)
	if err != nil {
		return err
	}

	defer fT.Close()

	if configInput.CassandraEnv.MallocArenaMax > 0 {
		if _, err := fmt.Fprintf(fT, "export MALLOC_ARENA_MAX=%d\n", configInput.CassandraEnv.MallocArenaMax); err != nil {
			return err
		}
	}

	if configInput.CassandraEnv.HeapDumpDir != "" {
		if _, err := fmt.Fprintf(fT, "export CASSANDRA_HEAPDUMP_DIR=%s\n", configInput.CassandraEnv.HeapDumpDir); err != nil {
			return err
		}
	}

	if _, err = fT.Write(f); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(fT, "\n"); err != nil {
		return err
	}

	for _, opt := range configInput.CassandraEnv.AdditionalOpts {
		if _, err := fmt.Fprintf(fT, "JVM_OPTS=\"$JVM_OPTS %s\"\n", opt); err != nil {
			return err
		}
	}

	return nil
}

// createJVMOptions writes all the jvm*-server.options
func createJVMOptions(configInput *ConfigInput, sourceDir, targetDir string) error {
	if err := createServerJVMOptions(configInput.ServerOptions, "jvm-server.options", sourceDir, targetDir); err != nil {
		return err
	}
	if err := createServerJVMOptions(configInput.ServerOptions11, "jvm11-server.options", sourceDir, targetDir); err != nil {
		return err
	}
	if err := createServerJVMOptions(configInput.ServerOptions17, "jvm17-server.options", sourceDir, targetDir); err != nil {
		return err
	}

	return nil
}

func optionsFilenameToMap(filename string) map[string]metadata.Metadata {
	switch filename {
	case "jvm-server.options":
		return gentypes.JvmServerOptionsPrefix
	case "jvm11-server.options":
		return gentypes.Jvm11ServerOptionsPrefix
	default:
		// JVM17 and newer do not have alias tables in the EDNs
		return make(map[string]metadata.Metadata, 0)
	}
}

func createServerJVMOptions(options map[string]interface{}, filename, sourceDir, targetDir string) error {
	// Read the current jvm-server-options as []string, do linear search to replace the values with the inputs we get
	optionsPath := filepath.Join(sourceDir, filename)
	currentOptions, err := readJvmServerOptions(optionsPath)
	if err != nil {
		return err
	}

	targetOptions := make([]string, 0, len(currentOptions)+len(options))

	if len(options) > 0 {
		// Parse the jvm-server-options
		if addOpts, found := options["additional-jvm-opts"]; found {
			// These should be appended..
			for _, v := range addOpts.([]interface{}) {
				targetOptions = append(targetOptions, v.(string))
			}
		}

		s := optionsFilenameToMap(filename)
		for k, v := range options {
			if k == "additional-jvm-opts" || k == "garbage_collector" {
				continue
			}

			if outputVal, found := s[k]; found {
				if outputVal.ValueType == metadata.TemplateValue {
					// We need another process here..
					continue
				}
				targetOptions = append(targetOptions, outputVal.Output(fmt.Sprintf("%v", v)))
			}
		}
	}

	if options == nil {
		options = make(map[string]interface{})
	}

	if filename == "jvm11-server.options" {
		if _, found := options["garbage_collector"]; !found {
			// This is only applicable to JVM11 options..
			options["garbage_collector"] = "G1GC"
		}

		if gcOpts, found := options["garbage_collector"]; found {
			// Get the GC options
			currentOptions = append(currentOptions, getGCOptions(fmt.Sprintf("%v", gcOpts), 11)...)
		}
	}

	// Add current options, if they're not there..
curOptions:
	for _, v := range currentOptions {
		curValueLoc := strings.Index(v, "=")

		for _, vT := range targetOptions {
			if suppressed, prefix := prefixMatcher(v); suppressed {
				if strings.HasPrefix(vT, prefix) {
					continue curOptions
				}
			}

			vc := v
			vTc := vT

			// Different value should not mean we can't compare
			targetValueLoc := strings.Index(vT, "=")
			if targetValueLoc > 0 && curValueLoc > 0 {
				vTc = vTc[:targetValueLoc]
				vc = vc[:curValueLoc]
			}

			if vc == vTc {
				continue curOptions
			}
		}
		targetOptions = append(targetOptions, v)
	}

	targetFileT := filepath.Join(targetDir, filename)
	fT, err := os.OpenFile(targetFileT, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0770)
	if err != nil {
		return err
	}

	for _, v := range targetOptions {
		_, err := fmt.Fprintf(fT, "%s\n", v)
		if err != nil {
			return err
		}
	}

	defer fT.Close()

	return nil
}

func getGCOptions(gcName string, jvmMajor int) []string {
	switch gcName {
	case "G1GC":
		return defaultG1Settings
	case "CMS":
		return defaultCMSSettings
	case "Shenandoah":
		return []string{"-XX:+UseShenandoahGC"}
	case "ZGC":
		zgcOpts := []string{"-XX:+UseZGC"}
		if jvmMajor < 17 {
			zgcOpts = append(zgcOpts, "-XX:+UnlockExperimentalVMOptions")
		}
		return zgcOpts
	default:
		// User needs to define all the settings
		return []string{}
	}
}

func prefixMatcher(value string) (bool, string) {
	// r := regexp.MustCompile(gentypes.JvmServerOptionsPrefixExp)
	parts := prefixRegexp.FindStringSubmatch(value)
	if parts != nil {
		return true, parts[0]
	}
	return false, ""
}

func readJvmServerOptions(path string) ([]string, error) {
	options := make([]string, 0)

	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return make([]string, 0), nil
		}
		return nil, err
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	runningSegment := false
	currentSegment := ""

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text()) // Avoid dual allocation from token -> string

		// Try to detect GC settings
		if strings.HasPrefix(line, "### ") {
			segmentName, _ := strings.CutPrefix(line, "### ")
			if !runningSegment {
				currentSegment = segmentName
				runningSegment = true
			} else {
				currentSegment = segmentName
			}
		}

		if !strings.HasPrefix(line, "#") && len(line) > 0 {
			if runningSegment && (strings.HasPrefix(currentSegment, "G1") || strings.HasPrefix(currentSegment, "CMS")) {
				continue
			}
			options = append(options, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return options, nil
}

// cassandra.yaml related functions

func createCassandraYaml(configInput *ConfigInput, nodeInfo *NodeInfo, sourceDir, targetDir string) error {
	targetConfigFileName := oldCassandraConfigName
	// Verify if we should use cassandra_latest.yaml (5.0 and newer) or cassandra.yaml (4.1 and older)
	if _, err := os.Stat(filepath.Join(sourceDir, latestCassandraConfigName)); err == nil {
		targetConfigFileName = latestCassandraConfigName
	}

	// Read the base file
	yamlPath := filepath.Join(sourceDir, targetConfigFileName)

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

	merged["listen_address"] = nodeInfo.ListenIP.String()
	merged["rpc_address"] = nodeInfo.RPCIP.String()
	delete(merged, "broadcast_address") // Sets it to the same as listen_address
	merged["broadcast_rpc_address"] = nodeInfo.BroadcastIP
	merged["endpoint_snitch"] = "GossipingPropertyFileSnitch"
	merged["cluster_name"] = configInput.ClusterInfo.Name

	return merged
}

func writeYaml(doc map[string]interface{}, targetFile string) error {
	b, err := yaml.Marshal(doc)
	if err != nil {
		return err
	}

	return os.WriteFile(targetFile, b, 0660)
}

func copyFiles(sourceDir, targetDir string) error {
	// Copy the files we're not modifying
	files := []string{"jvm-clients.options", "jvm11-clients.options", "jvm17-clients.options", "logback.xml", "logback-tools.xml"}

	for _, f := range files {
		sourceFile := filepath.Join(sourceDir, f)
		targetFile := filepath.Join(targetDir, f)

		if _, err := os.Stat(sourceFile); err == nil {
			if err := copyFile(sourceFile, targetFile); err != nil {
				return err
			}
		} else if !os.IsNotExist(err) {
			return err
		}
	}

	return nil
}

func copyFile(source, target string) error {
	src, err := os.Open(source)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to open %s", source))
	}
	defer src.Close()

	dst, err := os.Create(target)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to open %s", target))
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}
