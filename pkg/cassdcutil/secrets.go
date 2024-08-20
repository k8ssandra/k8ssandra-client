package cassdcutil

import (
	"context"
	"strings"

	corev1 "k8s.io/api/core/v1"

	cassdcapi "github.com/k8ssandra/cass-operator/apis/cassandra/v1beta1"
)

type CassandraAuth struct {
	Username string
	Password string

	KeystorePath       string
	KeystorePassword   string
	TruststorePath     string
	TruststorePassword string
}

// CassandraAuthDetails fetches the Cassandra superuser secrets for the given CassandraDatacenter.
func (c *CassManager) CassandraAuthDetails(ctx context.Context, cassdc *cassdcapi.CassandraDatacenter) (*CassandraAuth, error) {
	secret := &corev1.Secret{}
	if err := c.client.Get(ctx, cassdc.GetSuperuserSecretNamespacedName(), secret); err != nil {
		return nil, err
	}

	auth := &CassandraAuth{
		Username: string(secret.Data["username"]),
		Password: string(secret.Data["password"]),
	}

	if ClientEncryptionEnabled(cassdc) {
		encryptionOptions := SubSectionOfCassYaml(cassdc, "client_encryption_options")
		auth.KeystorePath = strings.TrimSpace(encryptionOptions["keystore"].Data().(string))
		auth.KeystorePassword = strings.TrimSpace(encryptionOptions["keystore_password"].Data().(string))
		auth.TruststorePath = strings.TrimSpace(encryptionOptions["truststore"].Data().(string))
		auth.TruststorePassword = strings.TrimSpace(encryptionOptions["truststore_password"].Data().(string))
	}

	return auth, nil
}
