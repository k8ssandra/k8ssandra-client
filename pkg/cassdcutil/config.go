package cassdcutil

import (
	"github.com/Jeffail/gabs/v2"
	cassdcapi "github.com/k8ssandra/cass-operator/apis/cassandra/v1beta1"
)

func ClientEncryptionEnabled(dc *cassdcapi.CassandraDatacenter) bool {
	config, err := gabs.ParseJSON(dc.Spec.Config)
	if err != nil {
		return false
	}

	if config.Exists("cassandra-yaml", "client_encryption_options") {
		if config.Path("cassandra-yaml.client_encryption_options.enabled").Data().(bool) {
			return true
		}
	}

	return false
}

func SubSectionOfCassYaml(dc *cassdcapi.CassandraDatacenter, section string) map[string]*gabs.Container {
	config, err := gabs.ParseJSON(dc.Spec.Config)
	if err != nil {
		return make(map[string]*gabs.Container)
	}

	if !config.Exists("cassandra-yaml") {
		return make(map[string]*gabs.Container)
	}

	return config.Path("cassandra-yaml").Path(section).ChildrenMap()
}
