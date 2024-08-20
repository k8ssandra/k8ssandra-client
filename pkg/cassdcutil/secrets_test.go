package cassdcutil

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	cassdcapi "github.com/k8ssandra/cass-operator/apis/cassandra/v1beta1"
)

func TestCassandraAuthDetails(t *testing.T) {
	scheme := runtime.NewScheme()
	assert := assert.New(t)
	assert.NoError(clientgoscheme.AddToScheme(scheme))
	assert.NoError(cassdcapi.AddToScheme(scheme))

	cassdc := &cassdcapi.CassandraDatacenter{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-dc",
		},
		Spec: cassdcapi.CassandraDatacenterSpec{
			ClusterName:         "test-cluster",
			SuperuserSecretName: "test-secret",
			Config:              json.RawMessage(clientEncryptionEnabled),
		},
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-secret",
		},
		Data: map[string][]byte{
			"username": []byte("test-cluster-superuser"),
			"password": []byte("cryptic-password"),
		},
	}

	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cassdc, secret).Build()
	cassManager := &CassManager{client: client}

	authDetails, err := cassManager.CassandraAuthDetails(context.TODO(), cassdc)
	assert.NoError(err)
	assert.NotNil(authDetails)

	assert.Equal("test-cluster-superuser", authDetails.Username)
	assert.Equal("cryptic-password", authDetails.Password)
	assert.Equal("/etc/encryption/node-keystore.jks", authDetails.KeystorePath)
	assert.Equal("dc2", authDetails.KeystorePassword)
	assert.Equal("/etc/encryption/node-keystore.jks", authDetails.TruststorePath)
	assert.Equal("dc2", authDetails.TruststorePassword)
}
