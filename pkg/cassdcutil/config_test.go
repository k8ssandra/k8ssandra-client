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
