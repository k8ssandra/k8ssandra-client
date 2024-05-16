package register

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	configapi "github.com/k8ssandra/k8ssandra-operator/apis/config/v1beta1"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestRegister(t *testing.T) {
	require := require.New(t)
	client1 := (*multiEnv)[0].GetClient("source-namespace")
	client2 := (*multiEnv)[1].GetClient("dest-namespace")

	if err := client1.Create((*multiEnv)[0].Context, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "source-namespace"}}); err != nil {
		require.NoError(err)
	}

	if err := client2.Create((*multiEnv)[1].Context, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "dest-namespace"}}); err != nil {
		require.NoError(err)
	}

	buildDir := os.Getenv("BUILD_DIR")
	if buildDir == "" {
		_, b, _, _ := runtime.Caller(0)
		buildDir = filepath.Join(filepath.Dir(b), "../../../build")
	}

	if _, err := os.Stat(buildDir); os.IsNotExist(err) {
		if err := os.Mkdir(buildDir, os.ModePerm); err != nil {
			require.NoError(err)
		}
	}

	kc1, err := (*multiEnv)[0].GetKubeconfig(t)
	if err != nil {
		require.NoError(err)

	}
	f, err := os.Create(buildDir + "/kubeconfig1")
	t.Cleanup(func() {
		require.NoError(f.Close())
		require.NoError(os.RemoveAll(buildDir + "/kubeconfig1"))
	})
	if err != nil {
		require.NoError(err)
	}
	t.Cleanup(func() {
		require.NoError(f.Close())
		require.NoError(os.RemoveAll(buildDir + "/kubeconfig2"))
	})
	if _, err := f.Write(kc1); err != nil {
		require.NoError(err)
	}

	kc2, err := (*multiEnv)[1].GetKubeconfig(t)
	if err != nil {
		require.NoError(err)

	}
	f, err = os.Create(buildDir + "/kubeconfig2")
	if err != nil {
		require.NoError(err)
	}
	t.Cleanup(func() {
		require.NoError(f.Close())
		require.NoError(os.RemoveAll(buildDir + "/kubeconfig2"))
	})
	if _, err := f.Write(kc2); err != nil {
		require.NoError(err)
	}
	ex := RegistrationExecutor{
		SourceKubeconfig: buildDir + "/kubeconfig1",
		DestKubeconfig:   buildDir + "/kubeconfig2",
		SourceContext:    "default-context",
		DestContext:      "default-context",
		SourceNamespace:  "source-namespace",
		DestNamespace:    "dest-namespace",
		ServiceAccount:   "k8ssandra-operator",
		Context:          context.TODO(),
		DestinationName:  "test-destination",
	}
	ctx := context.Background()

	require.Eventually(func() bool {
		res := ex.RegisterCluster()
		switch {
		case res.IsDone():
			return true
		case res.IsError():
			t.Log(res.GetError())
			if res.GetError().Error() == "no secret found for service account k8ssandra-operator" {
				return true
			}
		}
		return false
	}, time.Second*30, time.Second*5)

	// This relies on a controller that is not running in the envtest.

	desiredSaSecret := &corev1.Secret{}
	require.NoError(client1.Get(context.Background(), client.ObjectKey{Name: "k8ssandra-operator-secret", Namespace: "source-namespace"}, desiredSaSecret))
	patch := client.MergeFrom(desiredSaSecret.DeepCopy())
	desiredSaSecret.Data = map[string][]byte{
		"token":  []byte("test-token"),
		"ca.crt": []byte("test-ca"),
	}
	require.NoError(client1.Patch(ctx, desiredSaSecret, patch))

	desiredSa := &corev1.ServiceAccount{}
	require.NoError(client1.Get(
		context.Background(),
		client.ObjectKey{Name: "k8ssandra-operator", Namespace: "source-namespace"},
		desiredSa))

	patch = client.MergeFrom(desiredSa.DeepCopy())
	desiredSa.Secrets = []corev1.ObjectReference{
		{
			Name: "k8ssandra-operator-secret",
		},
	}
	require.NoError(client1.Patch(ctx, desiredSa, patch))

	// Continue reconciliation

	require.Eventually(func() bool {
		res := ex.RegisterCluster()
		switch {
		case res.IsDone():
			return true
		case res.IsError():
			t.Log(res.GetError())
			return false
		}
		return false
	}, time.Second*300, time.Second*1)

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
	}, time.Second*60, time.Second*5)

	destKubeconfig := ClientConfigFromSecret(destSecret)
	require.Equal(t,
		desiredSaSecret.Data["ca.crt"],
		destKubeconfig.Clusters["cluster"].CertificateAuthorityData)

	require.Equal(t,
		string(desiredSaSecret.Data["token"]),
		destKubeconfig.AuthInfos["cluster"].Token)
}

func ClientConfigFromSecret(s *corev1.Secret) clientcmdapi.Config {
	out, err := clientcmd.Load(s.Data["kubeconfig"])
	if err != nil {
		panic(err)
	}
	return *out
}
