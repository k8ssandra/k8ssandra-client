package register

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	configapi "github.com/k8ssandra/k8ssandra-operator/apis/config/v1beta1"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestRegister(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	deferFunc := startKind()
	defer deferFunc()

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

	// Test configuration
	saName := "k8ssandra-operator"
	releaseName := "k8ssandra-operator"

	// Create ClusterRoles that the ClusterRoleBindings will reference
	k8ssandraClusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-k8ssandra-operator", releaseName),
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"configmaps", "secrets", "services"},
				Verbs:     []string{"get", "list", "watch", "create", "update", "patch", "delete"},
			},
		},
	}
	require.NoError(client1.Create(ctx, k8ssandraClusterRole))

	cassClusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-cass-operator", releaseName),
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"pods", "services"},
				Verbs:     []string{"get", "list", "watch"},
			},
		},
	}
	require.NoError(client1.Create(ctx, cassClusterRole))

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
		ServiceAccount:   saName,
		ReleaseName:      releaseName,
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

	destKubeconfig, err := ClientConfigFromSecret(destSecret)
	require.NoError(err)
	require.Equal(
		sourceSecret.Data["ca.crt"],
		destKubeconfig.Clusters["test-destination"].CertificateAuthorityData)

	require.Equal(
		string(sourceSecret.Data["token"]),
		destKubeconfig.AuthInfos["test-destination"].Token)

	// Verify ClusterRoleBindings were created
	// ClusterRoleBinding name pattern: {ServiceAccount}-k8ssandra-operator
	// ClusterRole reference pattern: {ReleaseName}-k8ssandra-operator
	k8ssandraBinding := &rbacv1.ClusterRoleBinding{}
	require.NoError(client1.Get(ctx,
		client.ObjectKey{Name: fmt.Sprintf("%s-k8ssandra-operator", saName)},
		k8ssandraBinding))
	require.Equal(fmt.Sprintf("%s-k8ssandra-operator", releaseName), k8ssandraBinding.RoleRef.Name)
	require.Equal("ClusterRole", k8ssandraBinding.RoleRef.Kind)
	require.Len(k8ssandraBinding.Subjects, 1)
	require.Equal(saName, k8ssandraBinding.Subjects[0].Name)
	require.Equal("source-namespace", k8ssandraBinding.Subjects[0].Namespace)

	cassBinding := &rbacv1.ClusterRoleBinding{}
	require.NoError(client1.Get(ctx,
		client.ObjectKey{Name: fmt.Sprintf("%s-cass-operator", saName)},
		cassBinding))
	require.Equal(fmt.Sprintf("%s-cass-operator", releaseName), cassBinding.RoleRef.Name)
	require.Equal("ClusterRole", cassBinding.RoleRef.Kind)
	require.Len(cassBinding.Subjects, 1)
	require.Equal(saName, cassBinding.Subjects[0].Name)
	require.Equal("source-namespace", cassBinding.Subjects[0].Namespace)
}

func ClientConfigFromSecret(s *corev1.Secret) (clientcmdapi.Config, error) {
	out, err := clientcmd.Load(s.Data["kubeconfig"])
	if err != nil {
		return clientcmdapi.Config{}, err
	}
	return *out, nil
}

// TestRegisterIdempotency tests that re-registration works correctly (idempotency)
func TestRegisterIdempotency(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	deferFunc := startKind()
	defer deferFunc()

	require := require.New(t)
	client1, _ := client.New((*multiEnv)[0].RestConfig(), client.Options{})
	client2, _ := client.New((*multiEnv)[1].RestConfig(), client.Options{})
	ctx := context.Background()

	// Create namespaces
	require.Eventually(func() bool {
		err1 := client1.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "idempotency-source"}})
		if err1 != nil && !k8serrors.IsAlreadyExists(err1) {
			return false
		}
		err2 := client2.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "idempotency-dest"}})
		if err2 != nil && !k8serrors.IsAlreadyExists(err2) {
			return false
		}
		return true
	}, time.Second*6, time.Millisecond*100)

	// Create ClusterRoles
	k8ssandraClusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "idempotency-k8ssandra-operator",
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"configmaps"},
				Verbs:     []string{"get", "list"},
			},
		},
	}
	require.NoError(client1.Create(ctx, k8ssandraClusterRole))

	cassClusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "idempotency-cass-operator",
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"pods"},
				Verbs:     []string{"get", "list"},
			},
		},
	}
	require.NoError(client1.Create(ctx, cassClusterRole))

	// Create kubeconfig files
	f1, err := os.Create(testDir + "/kubeconfig-idempotency1")
	require.NoError(err)
	t.Cleanup(func() {
		require.NoError(f1.Close())
	})
	kc1, err := (*multiEnv)[0].GetKubeconfig()
	require.NoError(err)
	_, err = f1.Write(kc1)
	require.NoError(err)

	f2, err := os.Create(testDir + "/kubeconfig-idempotency2")
	require.NoError(err)
	t.Cleanup(func() {
		require.NoError(f2.Close())
	})
	kc2, err := (*multiEnv)[1].GetKubeconfig()
	require.NoError(err)
	_, err = f2.Write(kc2)
	require.NoError(err)

	ex := RegistrationExecutor{
		SourceKubeconfig: testDir + "/kubeconfig-idempotency1",
		DestKubeconfig:   testDir + "/kubeconfig-idempotency2",
		SourceContext:    "default-context",
		DestContext:      "default-context",
		SourceNamespace:  "idempotency-source",
		DestNamespace:    "idempotency-dest",
		ServiceAccount:   "test-sa",
		ReleaseName:      "idempotency",
		Context:          ctx,
		DestinationName:  "idempotency-test",
	}

	// First registration
	require.Eventually(func() bool {
		res := ex.RegisterCluster()
		return res == nil
	}, time.Second*6, time.Millisecond*100)

	// Verify ClusterRoleBindings were created
	k8ssandraBinding := &rbacv1.ClusterRoleBinding{}
	require.NoError(client1.Get(ctx,
		client.ObjectKey{Name: "test-sa-k8ssandra-operator"},
		k8ssandraBinding))

	// Second registration (should be idempotent)
	require.Eventually(func() bool {
		res := ex.RegisterCluster()
		return res == nil
	}, time.Second*6, time.Millisecond*100)

	// Verify ClusterRoleBindings still exist and are correct
	k8ssandraBinding2 := &rbacv1.ClusterRoleBinding{}
	require.NoError(client1.Get(ctx,
		client.ObjectKey{Name: "test-sa-k8ssandra-operator"},
		k8ssandraBinding2))
	require.Equal("idempotency-k8ssandra-operator", k8ssandraBinding2.RoleRef.Name)
	require.Equal("test-sa", k8ssandraBinding2.Subjects[0].Name)
}

// TestRegisterMissingClusterRole tests that registration continues when ClusterRoles don't exist
func TestRegisterMissingClusterRole(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	deferFunc := startKind()
	defer deferFunc()

	require := require.New(t)
	client1, _ := client.New((*multiEnv)[0].RestConfig(), client.Options{})
	client2, _ := client.New((*multiEnv)[1].RestConfig(), client.Options{})
	ctx := context.Background()

	// Create namespaces
	require.Eventually(func() bool {
		err1 := client1.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "missing-cr-source"}})
		if err1 != nil && !k8serrors.IsAlreadyExists(err1) {
			return false
		}
		err2 := client2.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "missing-cr-dest"}})
		if err2 != nil && !k8serrors.IsAlreadyExists(err2) {
			return false
		}
		return true
	}, time.Second*6, time.Millisecond*100)

	// Note: NOT creating ClusterRoles - they should be missing

	// Create kubeconfig files
	f1, err := os.Create(testDir + "/kubeconfig-missing-cr1")
	require.NoError(err)
	t.Cleanup(func() {
		require.NoError(f1.Close())
	})
	kc1, err := (*multiEnv)[0].GetKubeconfig()
	require.NoError(err)
	_, err = f1.Write(kc1)
	require.NoError(err)

	f2, err := os.Create(testDir + "/kubeconfig-missing-cr2")
	require.NoError(err)
	t.Cleanup(func() {
		require.NoError(f2.Close())
	})
	kc2, err := (*multiEnv)[1].GetKubeconfig()
	require.NoError(err)
	_, err = f2.Write(kc2)
	require.NoError(err)

	ex := RegistrationExecutor{
		SourceKubeconfig: testDir + "/kubeconfig-missing-cr1",
		DestKubeconfig:   testDir + "/kubeconfig-missing-cr2",
		SourceContext:    "default-context",
		DestContext:      "default-context",
		SourceNamespace:  "missing-cr-source",
		DestNamespace:    "missing-cr-dest",
		ServiceAccount:   "test-sa-missing",
		ReleaseName:      "missing-release",
		Context:          ctx,
		DestinationName:  "missing-cr-test",
	}

	// Registration should succeed even without ClusterRoles
	require.Eventually(func() bool {
		res := ex.RegisterCluster()
		return res == nil
	}, time.Second*6, time.Millisecond*100)

	// Verify ServiceAccount was created
	sa := &corev1.ServiceAccount{}
	require.NoError(client1.Get(ctx,
		client.ObjectKey{Name: "test-sa-missing", Namespace: "missing-cr-source"},
		sa))

	// Verify ClusterRoleBindings were NOT created (since ClusterRoles don't exist)
	k8ssandraBinding := &rbacv1.ClusterRoleBinding{}
	err = client1.Get(ctx,
		client.ObjectKey{Name: "test-sa-missing-k8ssandra-operator"},
		k8ssandraBinding)
	require.True(k8serrors.IsNotFound(err), "ClusterRoleBinding should not exist when ClusterRole is missing")

	// Verify destination resources were still created
	if err := configapi.AddToScheme(client2.Scheme()); err != nil {
		require.NoError(err)
	}
	destSecret := &corev1.Secret{}
	require.Eventually(func() bool {
		err = client2.Get(ctx,
			client.ObjectKey{Name: "missing-cr-test", Namespace: "missing-cr-dest"}, destSecret)
		return err == nil
	}, time.Second*6, time.Millisecond*100)
}
