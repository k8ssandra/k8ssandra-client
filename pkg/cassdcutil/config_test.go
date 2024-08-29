package cassdcutil

import (
	"encoding/json"
	"testing"

	cassdcapi "github.com/k8ssandra/cass-operator/apis/cassandra/v1beta1"
	"github.com/stretchr/testify/assert"
)

var clientEncryptionEnabled = `
{
	"cassandra-yaml": {
	  "client_encryption_options": {
	  	"enabled": true,
		"optional": false,
        "keystore": "/etc/encryption/node-keystore.jks",
        "keystore_password": "dc2",
        "truststore": "/etc/encryption/node-keystore.jks",
        "truststore_password": "dc2"
	  }
	}
}
`

func TestClientEncryptionEnabled(t *testing.T) {
	dc := &cassdcapi.CassandraDatacenter{
		Spec: cassdcapi.CassandraDatacenterSpec{
			Config: json.RawMessage(clientEncryptionEnabled),
		},
	}

	assert := assert.New(t)
	assert.True(ClientEncryptionEnabled(dc))
}

func TestEmptySubSection(t *testing.T) {
	dc := &cassdcapi.CassandraDatacenter{
		Spec: cassdcapi.CassandraDatacenterSpec{},
	}

	assert := assert.New(t)
	section := SubSectionOfCassYaml(dc, "client_encryption_options")
	assert.NotNil(section)
	assert.Equal(0, len(section))

	dc.Spec.Config = json.RawMessage(``)
	section = SubSectionOfCassYaml(dc, "client_encryption_options")
	assert.NotNil(section)
	assert.Equal(0, len(section))
}

func TestSubSectionNotMatch(t *testing.T) {
	dc := &cassdcapi.CassandraDatacenter{
		Spec: cassdcapi.CassandraDatacenterSpec{
			Config: json.RawMessage(clientEncryptionEnabled),
		},
	}

	assert := assert.New(t)
	section := SubSectionOfCassYaml(dc, "server_encryption_options")
	assert.NotNil(section)
	assert.Equal(0, len(section))
}

func TestSubSectionPart(t *testing.T) {
	dc := &cassdcapi.CassandraDatacenter{
		Spec: cassdcapi.CassandraDatacenterSpec{
			Config: json.RawMessage(clientEncryptionEnabled),
		},
	}

	assert := assert.New(t)
	section := SubSectionOfCassYaml(dc, "client_encryption_options")
	assert.NotNil(section)
	assert.Equal(6, len(section))

	enabled, ok := section["enabled"].Data().(bool)
	assert.True(ok)
	assert.True(enabled)

	keystore, ok := section["keystore"].Data().(string)
	assert.True(ok)
	assert.Equal("/etc/encryption/node-keystore.jks", keystore)

	keystorePassword, ok := section["keystore_password"].Data().(string)
	assert.True(ok)
	assert.Equal("dc2", keystorePassword)

	truststore, ok := section["truststore"].Data().(string)
	assert.True(ok)
	assert.Equal("/etc/encryption/node-keystore.jks", truststore)

	truststorePassword, ok := section["truststore_password"].Data().(string)
	assert.True(ok)
	assert.Equal("dc2", truststorePassword)

	optional, ok := section["optional"].Data().(bool)
	assert.True(ok)
	assert.False(optional)
}
