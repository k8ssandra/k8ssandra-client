// Code is generated with scripts/parse.py. DO NOT EDIT. Sadly, the script is lost in time..
package metadata

import(
	"regexp"
)

var jvm_server_options = map[string]Metadata{
	"-XX:+UnlockDiagnosticVMOptions":                       {Key: "unlock-diagnostic-vm-options", BuilderType: "boolean", DefaultValueString: "true"},
	"-Dcassandra.available_processors":                     {Key: "cassandra_available_processors", BuilderType: "int"},
	"-Dcassandra.config":                                   {Key: "cassandra_config_directory", BuilderType: "string"},
	"-Dcassandra.initial_token":                            {Key: "cassandra_initial_token", BuilderType: "string"},
	"-Dcassandra.join_ring":                                {Key: "cassandra_join_ring", BuilderType: "boolean", DefaultValueString: "true"},
	"-Dcassandra.load_ring_state":                          {Key: "cassandra_load_ring_state", BuilderType: "boolean", DefaultValueString: "true"},
	"-Dcassandra.metricsReporterConfigFile":                {Key: "cassandra_metrics_reporter_config_file", BuilderType: "string"},
	"-Dcassandra.replace_address":                          {Key: "cassandra_replace_address", BuilderType: "string"},
	"-Ddse.consistent_replace":                             {Key: "dse_consistent_replace", BuilderType: "string"},
	"-Ddse.consistent_replace.parallelism":                 {Key: "dse_consistent_replace_parallelism", BuilderType: "string"},
	"-Ddse.consistent_replace.retries":                     {Key: "dse_consistent_replace_retries", BuilderType: "string"},
	"-Ddse.consistent_replace.whitelist":                   {Key: "dse_consistent_replace_whitelist", BuilderType: "string"},
	"-Dcassandra.replayList":                               {Key: "cassandra_replay_list", BuilderType: "string"},
	"-Dcassandra.ring_delay_ms":                            {Key: "cassandra_ring_delay_ms", BuilderType: "int"},
	"-Dcassandra.triggers_dir":                             {Key: "cassandra_triggers_dir", BuilderType: "string"},
	"-Dcassandra.write_survey":                             {Key: "cassandra_write_survey", BuilderType: "boolean", DefaultValueString: "false"},
	"-Dcassandra.disable_auth_caches_remote_configuration": {Key: "cassandra_disable_auth_caches_remote_configuration", BuilderType: "boolean", DefaultValueString: "false"},
	"-Dcassandra.force_default_indexing_page_size":         {Key: "cassandra_force_default_indexing_page_size", BuilderType: "boolean", DefaultValueString: "false"},
	"-Dcassandra.maxHintTTL":                               {Key: "cassandra_max_hint_ttl", BuilderType: "string"},
	"-XX:+UseThreadPriorities":                             {Key: "use_thread_priorities", BuilderType: "boolean", DefaultValueString: "true"},
	"-XX:+HeapDumpOnOutOfMemoryError":                      {Key: "heap_dump_on_out_of_memory_error", BuilderType: "boolean", DefaultValueString: "true"},
	"-Xss":                                                 {Key: "per_thread_stack_size", BuilderType: "string", DefaultValueString: "256k"},
	"-XX:StringTableSize":                                  {Key: "string_table_size", BuilderType: "string", DefaultValueString: "1000003"},
	"-XX:+AlwaysPreTouch":                                  {Key: "always_pre_touch", BuilderType: "boolean", DefaultValueString: "true"},
	"-XX:+UseTLAB":                                         {Key: "use_tlb", BuilderType: "boolean", DefaultValueString: "true"},
	"-XX:+ResizeTLAB":                                      {Key: "resize_tlb", BuilderType: "boolean", DefaultValueString: "true"},
	"-XX:+UseNUMA":                                         {Key: "use_numa", BuilderType: "boolean", DefaultValueString: "true"},
	"-XX:+PerfDisableSharedMem":                            {Key: "perf_disable_shared_mem", BuilderType: "boolean", DefaultValueString: "true"},
	"-Djava.net.preferIPv4Stack":                           {Key: "java_net_prefer_ipv4_stack", BuilderType: "boolean", DefaultValueString: "true"},
	"-Dsun.nio.PageAlignDirectMemory":                      {Key: "page-align-direct-memory", BuilderType: "boolean", DefaultValueString: "true"},
	"-XX:-RestrictContended":                               {Key: "restrict-contended", BuilderType: "boolean", DefaultValueString: "true"},
	"-XX:GuaranteedSafepointInterval":                      {Key: "guaranteed-safepoint-interval", BuilderType: "string", DefaultValueString: "300000"},
	"-XX:-UseBiasedLocking":                                {Key: "use-biased-locking", BuilderType: "boolean", DefaultValueString: "true"},
	"-XX:+DebugNonSafepoints":                              {Key: "debug-non-safepoints", BuilderType: "boolean", DefaultValueString: "true"},
	"-XX:+PreserveFramePointer":                            {Key: "preserve-frame-pointer", BuilderType: "boolean", DefaultValueString: "true"},
	"-XX:+UnlockCommercialFeatures":                        {Key: "unlock_commercial_features", BuilderType: "boolean", DefaultValueString: "false"},
	"-XX:+FlightRecorder":                                  {Key: "flight_recorder", BuilderType: "boolean", DefaultValueString: "false"},
	"-XX:+LogCompilation":                                  {Key: "log_compilation", BuilderType: "boolean", DefaultValueString: "false"},
	"-Xms":                                                 {Key: "initial_heap_size", BuilderType: "string"},
	"-Xmx":                                                 {Key: "max_heap_size", BuilderType: "string"},
	"-Djdk.nio.maxCachedBufferSize":                        {Key: "jdk_nio_maxcachedbuffersize", BuilderType: "int", DefaultValueString: "1048576"},
	"-Dcassandra.expiration_date_overflow_policy":          {Key: "cassandra_expiration_date_overflow_policy", BuilderType: "string"},
	"-Dio.netty.eventLoop.maxPendingTasks":                 {Key: "io_netty_eventloop_maxpendingtasks", BuilderType: "int", DefaultValueString: "65536"},
	"-XX:+CrashOnOutOfMemoryError":                         {Key: "crash_on_out_of_memory_error", BuilderType: "boolean", DefaultValueString: "false"},
	"-XX:MaxDirectMemorySize":                              {Key: "max_direct_memory", BuilderType: "string"},
	"-Dcassandra.printHeapHistogramOnOutOfMemoryError":     {Key: "print_heap_histogram_on_out_of_memory_error", BuilderType: "boolean", DefaultValueString: "false"},
	"-XX:+ExitOnOutOfMemoryError":                          {Key: "exit_on_out_of_memory_error", BuilderType: "boolean", DefaultValueString: "false"},
}

const (
	jvm_server_optionsPrefixExp = "^-Xss|^-Xms|^-Xmx"
)

var (
	prefixMatcher = regexp.MustCompile(jvm_server_optionsPrefixExp)
)

func ServerOptions() map[string]string {
	// Inverses the above map until we can regenerate something nicer..
	m := make(map[string]string, len(jvm_server_options))
	for k, v := range jvm_server_options {
		m[v.Key] = k
	}
	return m
}

func PrefixParser(input string) (bool, string) {
	// prefixMatcher := regexp.MustCompile(jvm_server_optionsPrefixExp)
	subParts := prefixMatcher.FindStringSubmatchIndex(input)
	if len(subParts) > 0 {
		mapKey := input[subParts[0]:subParts[1]]

		return true, mapKey
	}

	return false, ""
}

type Metadata struct {
	// omitEmpty bool // static_constant vs constant
	Key                string
	BuilderType        string // list, boolean, string, int => yaml rendering
	DefaultValueString string
	// DefaultValueInt    int
	// DefaultValueBool   bool
}
