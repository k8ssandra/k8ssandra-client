package register

import (
	"context"
	"os"
	"testing"
	"time"

	configapi "github.com/k8ssandra/k8ssandra-operator/apis/config/v1beta1"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestRegister(t *testing.T) {
	require := require.New(t)
	client1, _ := client.New((*multiEnv)[0].RestConfig(), client.Options{})
	client2, _ := client.New((*multiEnv)[1].RestConfig(), client.Options{})
	ctx := context.Background()
	require.Eventually(func() bool {
		// It seems that at first, these clients may not be ready for use. By the time they can create a namespace they are known ready.
		err1 := client1.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "source-namespace"}})
		if err1 != nil {
			t.Log(err1)
			if k8serrors.IsAlreadyExists(err1) {
				err1 = nil
			}
		}
		err2 := client2.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "dest-namespace"}})
		if err2 != nil {
			t.Log(err2)
			if k8serrors.IsAlreadyExists(err2) {
				err2 = nil
			}
		}
		return err1 == nil && err2 == nil
	}, time.Second*6, time.Millisecond*100)

	f1, err := os.Create(testDir + "/kubeconfig1")
	require.NoError(err)
	t.Cleanup(func() {
		require.NoError(f1.Close())
	})
	kc1, err := (*multiEnv)[0].GetKubeconfig()
	require.NoError(err)
	_, err = f1.Write(kc1)
	require.NoError(err)

	f2, err := os.Create(testDir + "/kubeconfig2")
	require.NoError(err)
	t.Cleanup(func() {
		require.NoError(f2.Close())
	})

	kc2, err := (*multiEnv)[1].GetKubeconfig()
	require.NoError(err)
	_, err = f2.Write(kc2)
	require.NoError(err)

	ex := RegistrationExecutor{
		SourceKubeconfig: testDir + "/kubeconfig1",
		DestKubeconfig:   testDir + "/kubeconfig2",
		SourceContext:    "default-context",
		DestContext:      "default-context",
		SourceNamespace:  "source-namespace",
		DestNamespace:    "dest-namespace",
		ServiceAccount:   "k8ssandra-operator",
		Context:          ctx,
		DestinationName:  "test-destination",
	}
	// Continue reconciliation
	require.Eventually(func() bool {
		res := ex.RegisterCluster()
		return res == nil
	}, time.Second*6, time.Millisecond*100)

	sourceSecret := &corev1.Secret{}
	// Ensure secret created.
	require.Eventually(func() bool {
		err := client1.Get(ctx, types.NamespacedName{Name: "k8ssandra-operator-secret", Namespace: "source-namespace"}, sourceSecret)
		return err == nil
	}, time.Second*6, time.Millisecond*100)

	desiredSa := &corev1.ServiceAccount{}
	require.NoError(client1.Get(
		context.Background(),
		client.ObjectKey{Name: "k8ssandra-operator", Namespace: "source-namespace"},
		desiredSa))

	if err := configapi.AddToScheme(client2.Scheme()); err != nil {
		require.NoError(err)
	}
	destSecret := &corev1.Secret{}
	require.Eventually(func() bool {
		err = client2.Get(ctx,
			client.ObjectKey{Name: "test-destination", Namespace: "dest-namespace"}, destSecret)
		if err != nil {
			t.Log("didn't find dest secret")
			return false
		}
		clientConfig := &configapi.ClientConfig{}
		err = client2.Get(ctx,
			client.ObjectKey{Name: "test-destination", Namespace: "dest-namespace"}, clientConfig)
		if err != nil {
			t.Log("didn't find dest client config")
			return false
		}
		return err == nil
	}, time.Second*6, time.Millisecond*100)

	destKubeconfig := ClientConfigFromSecret(destSecret)
	require.Equal(
		sourceSecret.Data["ca.crt"],
		destKubeconfig.Clusters["test-destination"].CertificateAuthorityData)

	require.Equal(
		string(sourceSecret.Data["token"]),
		destKubeconfig.AuthInfos["test-destination"].Token)
}

func ClientConfigFromSecret(s *corev1.Secret) clientcmdapi.Config {
	out, err := clientcmd.Load(s.Data["kubeconfig"])
	if err != nil {
		panic(err)
	}
	return *out
}
