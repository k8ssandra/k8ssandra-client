package register

import (
	"context"
	"fmt"

	"github.com/charmbracelet/log"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/k8ssandra/k8ssandra-client/pkg/registration"
	configapi "github.com/k8ssandra/k8ssandra-operator/apis/config/v1beta1"
)

type RegistrationExecutor struct {
	DestinationName    string
	SourceKubeconfig   string
	DestKubeconfig     string
	SourceContext      string
	DestContext        string
	SourceNamespace    string
	DestNamespace      string
	ServiceAccount     string
	ReleaseName        string
	OverrideSourceIP   string
	OverrideSourcePort string
	Context            context.Context
}

func getDefaultSecret(saNamespace, saName string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      saName + "-secret",
			Namespace: saNamespace,
			Annotations: map[string]string{
				"kubernetes.io/service-account.name": saName,
			},
		},
		Type: corev1.SecretTypeServiceAccountToken,
	}
}

func getDefaultServiceAccount(saName, saNamespace string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      saName,
			Namespace: saNamespace,
		},
	}
}

func (e *RegistrationExecutor) RegisterCluster() error {
	log.Printf("Registering cluster from context: %s to context: %s", e.SourceContext, e.DestContext)

	if e.SourceContext == e.DestContext && e.SourceKubeconfig == e.DestKubeconfig {
		return NonRecoverable("source and destination context and kubeconfig are the same, you should not register the same cluster to itself. Reference it by leaving the k8sContext field blank instead")
	}

	srcClient, err := registration.GetClient(e.SourceKubeconfig, e.SourceContext)
	if err != nil {
		return err
	}

	destClient, err := registration.GetClient(e.DestKubeconfig, e.DestContext)
	if err != nil {
		return err
	}

	// Get ServiceAccount
	serviceAccount := &corev1.ServiceAccount{}
	if err := srcClient.Get(e.Context, client.ObjectKey{Name: e.ServiceAccount, Namespace: e.SourceNamespace}, serviceAccount); err != nil {
		if apierrors.IsNotFound(err) {
			if err := srcClient.Create(e.Context, getDefaultServiceAccount(e.ServiceAccount, e.SourceNamespace)); err != nil {
				return err
			}
		}
		return err
	}

	// Create ClusterRoleBindings only when registering across different namespaces
	// When namespaces match, Helm-installed ClusterRoleBindings are sufficient
	if e.SourceNamespace != e.DestNamespace {
		log.Debug("Creating ClusterRoleBindings for cross-namespace registration",
			"sourceNamespace", e.SourceNamespace,
			"destNamespace", e.DestNamespace)
		if err := e.createClusterRoleBindings(srcClient); err != nil {
			log.Warn("Failed to create ClusterRoleBindings", "error", err)
			// Continue anyway - bindings might be managed externally or ClusterRoles might not exist yet
		}
	}

	// Get a secret in this namespace which holds the service account token
	secretsList := &corev1.SecretList{}
	if err := srcClient.List(e.Context, secretsList, client.InNamespace(e.SourceNamespace)); err != nil {
		return err
	}

	var secret *corev1.Secret
	for _, s := range secretsList.Items {
		if s.Annotations["kubernetes.io/service-account.name"] == e.ServiceAccount && s.Type == corev1.SecretTypeServiceAccountToken {
			secret = &s
			break
		}
	}

	if secret == nil {
		secret = getDefaultSecret(e.SourceNamespace, e.ServiceAccount)
		if err := srcClient.Create(e.Context, secret); err != nil {
			return err
		}
		return fmt.Errorf("no secret found for service account %s", e.ServiceAccount)
	}

	// Create Secret on destination cluster
	host, err := registration.KubeconfigToHost(e.SourceKubeconfig, e.SourceContext, e.OverrideSourceIP, e.OverrideSourcePort)
	if err != nil {
		return err
	}
	saConfig, err := registration.TokenToKubeconfig(*secret, host, e.DestinationName)
	if err != nil {
		return fmt.Errorf("error converting token to kubeconfig: %s, secret: %#v", err.Error(), secret)
	}

	secretData, err := clientcmd.Write(saConfig)
	if err != nil {
		return err
	}
	destSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      e.DestinationName,
			Namespace: e.DestNamespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"kubeconfig": secretData,
		},
	}
	if err := destClient.Create(e.Context, &destSecret); err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("error creating secret. err: %s sa %s", err, e.ServiceAccount)
	}

	// Create ClientConfig on destination cluster
	if err := configapi.AddToScheme(destClient.Scheme()); err != nil {
		return err
	}

	destClientConfig := configapi.ClientConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      e.DestinationName,
			Namespace: e.DestNamespace,
		},
		Spec: configapi.ClientConfigSpec{
			KubeConfigSecret: corev1.LocalObjectReference{
				Name: e.DestinationName,
			},
			ContextName: e.DestinationName,
		},
	}
	if err := destClient.Create(e.Context, &destClientConfig); err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

// createClusterRoleBindings creates ClusterRoleBindings for k8ssandra and cass operators.
// This grants the ServiceAccount permissions to manage Cassandra resources in the source cluster.
func (e *RegistrationExecutor) createClusterRoleBindings(srcClient client.Client) error {
	// Define the ClusterRoleBindings to create
	bindings := []struct {
		name        string
		clusterRole string
	}{
		{
			name:        fmt.Sprintf("%s-k8ssandra-operator", e.ServiceAccount),
			clusterRole: fmt.Sprintf("%s-k8ssandra-operator", e.ReleaseName),
		},
		{
			name:        fmt.Sprintf("%s-cass-operator", e.ServiceAccount),
			clusterRole: fmt.Sprintf("%s-cass-operator", e.ReleaseName),
		},
	}

	for _, binding := range bindings {
		if err := e.createOrUpdateClusterRoleBinding(srcClient, binding.name, binding.clusterRole); err != nil {
			return fmt.Errorf("failed to create ClusterRoleBinding %s: %w", binding.name, err)
		}
		log.Debug("ClusterRoleBinding created or updated", "name", binding.name, "clusterRole", binding.clusterRole)
	}

	return nil
}

// createOrUpdateClusterRoleBinding creates or updates a single ClusterRoleBinding.
// It uses an idempotent approach to handle re-registration scenarios.
func (e *RegistrationExecutor) createOrUpdateClusterRoleBinding(
	srcClient client.Client,
	bindingName string,
	clusterRoleName string,
) error {
	// First, check if the ClusterRole exists
	clusterRole := &rbacv1.ClusterRole{}
	if err := srcClient.Get(e.Context, client.ObjectKey{Name: clusterRoleName}, clusterRole); err != nil {
		if apierrors.IsNotFound(err) {
			log.Warn("ClusterRole not found, skipping ClusterRoleBinding creation",
				"clusterRole", clusterRoleName,
				"binding", bindingName)
			return nil // Non-fatal: ClusterRole might be created later
		}
		return fmt.Errorf("failed to check ClusterRole %s: %w", clusterRoleName, err)
	}

	// Create the ClusterRoleBinding object
	crb := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: bindingName,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     clusterRoleName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      e.ServiceAccount,
				Namespace: e.SourceNamespace,
			},
		},
	}

	// Try to create the ClusterRoleBinding
	if err := srcClient.Create(e.Context, crb); err != nil {
		if apierrors.IsAlreadyExists(err) {
			// ClusterRoleBinding already exists, update it to ensure it's correct
			existingCRB := &rbacv1.ClusterRoleBinding{}
			if err := srcClient.Get(e.Context, client.ObjectKey{Name: bindingName}, existingCRB); err != nil {
				return fmt.Errorf("failed to get existing ClusterRoleBinding %s: %w", bindingName, err)
			}

			// Update the existing ClusterRoleBinding
			existingCRB.RoleRef = crb.RoleRef
			existingCRB.Subjects = crb.Subjects
			if err := srcClient.Update(e.Context, existingCRB); err != nil {
				return fmt.Errorf("failed to update ClusterRoleBinding %s: %w", bindingName, err)
			}
			log.Debug("ClusterRoleBinding updated", "name", bindingName)
			return nil
		}
		return fmt.Errorf("failed to create ClusterRoleBinding %s: %w", bindingName, err)
	}

	log.Debug("ClusterRoleBinding created", "name", bindingName)
	return nil
}
