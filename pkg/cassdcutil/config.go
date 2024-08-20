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
		return nil
	}

	cassYaml := config.Path("cassandra-yaml")
	if cassYaml == nil {
		return make(map[string]*gabs.Container)
	}

	return cassYaml.Path(section).ChildrenMap()
}

/*
func (dc *CassandraDatacenter) LegacyInternodeEnabled() bool {
	config, err := gabs.ParseJSON(dc.Spec.Config)
	if err != nil {
		return false
	}

	hasOldKeyStore := func(gobContainer map[string]*gabs.Container) bool {
		if gobContainer == nil {
			return false
		}

		if keystorePath, found := gobContainer["keystore"]; found {
			if strings.TrimSpace(keystorePath.Data().(string)) == "/etc/encryption/node-keystore.jks" {
				return true
			}
		}
		return false
	}

	if config.Exists("cassandra-yaml", "client_encryption_options") || config.Exists("cassandra-yaml", "server_encryption_options") {
		serverContainer := config.Path("cassandra-yaml.server_encryption_options").ChildrenMap()
		clientContainer := config.Path("cassandra-yaml.client_encryption_options").ChildrenMap()

		if hasOldKeyStore(clientContainer) || hasOldKeyStore(serverContainer) {
			return true
		}
	}

	return false
}
*/
