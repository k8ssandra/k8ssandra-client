package registration

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetClient(configFileLocation string, contextName string) (client.Client, error) {
	clientConfig, err := clientcmd.LoadFromFile(GetKubeconfigFileLocation(configFileLocation))
	if err != nil {
		return nil, err
	}
	var restConfig *rest.Config
	if contextName == "" {
		restConfig, err := clientcmd.NewDefaultClientConfig(*clientConfig, &clientcmd.ConfigOverrides{}).ClientConfig()
		if err != nil {
			panic(err)
		}
		return client.New(restConfig, client.Options{})
	}

	context, found := clientConfig.Contexts[contextName]
	if !found {
		panic(errors.New(fmt.Sprint("context not found in supplied kubeconfig ", "contextName: ", contextName, " configFileLocation: ", GetKubeconfigFileLocation(configFileLocation))))
	}
	overrides := &clientcmd.ConfigOverrides{
		Context:     *context,
		ClusterInfo: *clientConfig.Clusters[context.Cluster],
		AuthInfo:    *clientConfig.AuthInfos[context.AuthInfo],
	}

	cConfig := clientcmd.NewNonInteractiveClientConfig(*clientConfig, contextName, overrides, clientcmd.NewDefaultClientConfigLoadingRules())
	restConfig, err = cConfig.ClientConfig()

	if err != nil {
		panic(err)
	}
	return client.New(restConfig, client.Options{})
}

func GetKubeconfigFileLocation(location string) string {
	if location != "" {
		return location
	} else if kubeconfigEnvVar, found := os.LookupEnv("KUBECONFIG"); found {
		return kubeconfigEnvVar
	} else {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			panic(err)
		}
		return filepath.Join(homeDir, ".kube", "config")
	}
}

func KubeconfigToHost(configFileLocation string, contextName string) (string, error) {
	clientConfig, err := clientcmd.LoadFromFile(GetKubeconfigFileLocation(configFileLocation))
	if err != nil {
		return "", err
	}
	if contextName == "" {
		restConfig, err := clientcmd.NewDefaultClientConfig(*clientConfig, &clientcmd.ConfigOverrides{}).ClientConfig()
		if err != nil {
			return "", err
		}
		return restConfig.Host, nil
	}

	context, found := clientConfig.Contexts[contextName]
	if !found {
		panic(errors.New(fmt.Sprint("context not found in supplied kubeconfig ", "contextName: ", contextName, " configFileLocation: ", GetKubeconfigFileLocation(configFileLocation))))
	}
	return clientConfig.Clusters[context.Cluster].Server, nil
}
