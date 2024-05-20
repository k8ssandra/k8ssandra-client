package register

import (
	"context"
	"fmt"

	"github.com/charmbracelet/log"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/k8ssandra/k8ssandra-client/pkg/registration"
	configapi "github.com/k8ssandra/k8ssandra-operator/apis/config/v1beta1"
)

type RegistrationExecutor struct {
	DestinationName  string
	SourceKubeconfig string
	DestKubeconfig   string
	SourceContext    string
	DestContext      string
	SourceNamespace  string
	DestNamespace    string
	ServiceAccount   string
	Context          context.Context
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
	log.Printf("Registering cluster from %s Context: %s to %s Context: %s",
		registration.GetKubeconfigFileLocation(e.SourceKubeconfig), e.SourceContext,
		registration.GetKubeconfigFileLocation(e.DestKubeconfig), e.DestContext,
	)
	if e.SourceContext == e.DestContext && e.SourceKubeconfig == e.DestKubeconfig {
		return NonRecoverableError{Message: "source and destination context and kubeconfig are the same, you should not register the same cluster to itself. Reference it by leaving the k8sContext field blank instead"}
	}
	srcClient, err := registration.GetClient(e.SourceKubeconfig, e.SourceContext)
	if err != nil {
		return RetryableError{Message: err.Error()}
	}
	destClient, err := registration.GetClient(e.DestKubeconfig, e.DestContext)
	if err != nil {
		return RetryableError{Message: err.Error()}
	}
	// Get ServiceAccount
	serviceAccount := &corev1.ServiceAccount{}
	if err := srcClient.Get(e.Context, client.ObjectKey{Name: e.ServiceAccount, Namespace: e.SourceNamespace}, serviceAccount); err != nil {
		if apierrors.IsNotFound(err) {
			if err := srcClient.Create(e.Context, getDefaultServiceAccount(e.ServiceAccount, e.SourceNamespace)); err != nil {
				return RetryableError{Message: err.Error()}
			}
		}
		return RetryableError{Message: err.Error()}
	}
	// Get a secret in this namespace which holds the service account token
	secretsList := &corev1.SecretList{}
	if err := srcClient.List(e.Context, secretsList, client.InNamespace(e.SourceNamespace)); err != nil {
		return RetryableError{Message: err.Error()}
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
			return RetryableError{Message: err.Error()}
		}
		return RetryableError{Message: fmt.Sprintf("no secret found for service account %s", e.ServiceAccount)}
	}

	// Create Secret on destination cluster
	host, err := registration.KubeconfigToHost(e.SourceKubeconfig, e.SourceContext)
	if err != nil {
		return RetryableError{Message: err.Error()}
	}
	saConfig, err := registration.TokenToKubeconfig(*secret, host)
	if err != nil {
		return RetryableError{fmt.Sprintf("error converting token to kubeconfig: %s, secret: %#v", err.Error(), secret)}
	}
	secretData, err := clientcmd.Write(saConfig)
	if err != nil {
		return RetryableError{Message: err.Error()}
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
	if err := destClient.Create(e.Context, &destSecret); err != nil {
		return RetryableError{fmt.Sprintf("error creating secret. err: %s sa %s", err, e.ServiceAccount)}
	}

	// Create ClientConfig on destination cluster
	if err := configapi.AddToScheme(destClient.Scheme()); err != nil {
		return RetryableError{Message: err.Error()}
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
		},
	}
	if err := destClient.Create(e.Context, &destClientConfig); err != nil {
		return RetryableError{Message: err.Error()}
	}
	return nil
}
