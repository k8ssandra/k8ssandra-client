package register

import (
	"context"
	"errors"
	"fmt"
	"log"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/k8ssandra/k8ssandra-client/pkg/registration"
	configapi "github.com/k8ssandra/k8ssandra-operator/apis/config/v1beta1"
	"github.com/k8ssandra/k8ssandra-operator/pkg/result"
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

func getDefaultSecret(saName, saNamespace string) *corev1.Secret {
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

func (e *RegistrationExecutor) RegisterCluster() result.ReconcileResult {
	log.Printf("Registering cluster from %s Context: %s to %s Context: %s",
		registration.GetKubeconfigFileLocation(e.SourceKubeconfig), e.SourceContext,
		registration.GetKubeconfigFileLocation(e.DestKubeconfig), e.DestContext,
	)
	if e.SourceContext == e.DestContext && e.SourceKubeconfig == e.DestKubeconfig {
		panic(errors.New("source and destination context and kubeconfig are the same, you should not register the same cluster to itself. Reference it by leaving the k8sContext field blank instead"))
	}
	srcClient, err := registration.GetClient(e.SourceKubeconfig, e.SourceContext)
	if err != nil {
		return result.Error(err)
	}
	destClient, err := registration.GetClient(e.DestKubeconfig, e.DestContext)
	if err != nil {
		return result.Error(err)
	}
	// Get ServiceAccount
	serviceAccount := &corev1.ServiceAccount{}
	if err := srcClient.Get(e.Context, client.ObjectKey{Name: e.ServiceAccount, Namespace: e.SourceNamespace}, serviceAccount); err != nil {
		if err := srcClient.Create(e.Context, getDefaultServiceAccount(e.ServiceAccount, e.SourceNamespace)); err != nil {
			return result.Error(err)
		}
		return result.Error(err)
	}
	// Get a secret in this namespace which holds the service account token
	secretsList := &corev1.SecretList{}
	if err := srcClient.List(e.Context, secretsList, client.InNamespace(e.SourceNamespace)); err != nil {
		return result.Error(err)
	}
	var secret *corev1.Secret
	for _, s := range secretsList.Items {
		if s.Annotations["kubernetes.io/service-account.name"] == e.ServiceAccount && s.Type == corev1.SecretTypeServiceAccountToken {
			secret = &s
			break
		}
	}
	if secret == nil {
		secret = getDefaultSecret(e.ServiceAccount, e.SourceNamespace)
		if err := srcClient.Create(e.Context, secret); err != nil {
			return result.Error(err)
		}
		return result.Error(fmt.Errorf("no secret found for service account %s", e.ServiceAccount))
	}

	// Create Secret on destination cluster
	host, err := registration.KubeconfigToHost(e.SourceKubeconfig, e.SourceContext)
	if err != nil {
		return result.Error(err)
	}
	saConfig, err := registration.TokenToKubeconfig(*secret, host)
	if err != nil {
		return result.Error(fmt.Errorf("error converting token to kubeconfig: %w, secret: %#v", err, secret))
	}
	secretData, err := clientcmd.Write(saConfig)
	if err != nil {
		return result.Error(err)
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
		return result.Error(fmt.Errorf("error creating secret. err: %s sa %s", err, e.ServiceAccount))
	}

	// Create ClientConfig on destination cluster
	if err := configapi.AddToScheme(destClient.Scheme()); err != nil {
		return result.Error(err)
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
		return result.Error(err)
	}
	return result.Done()
}
