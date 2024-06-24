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
	path, err := GetKubeconfigFileLocation(configFileLocation)
	if err != nil {
		return nil, err
	}
	clientConfig, err := clientcmd.LoadFromFile(path)
	if err != nil {
		return nil, err
	}
	var restConfig *rest.Config
	if contextName == "" {
		restConfig, err := clientcmd.NewDefaultClientConfig(*clientConfig, &clientcmd.ConfigOverrides{}).ClientConfig()
		if err != nil {
			return nil, err
		}
		return client.New(restConfig, client.Options{})
	}

	context, found := clientConfig.Contexts[contextName]
	if !found {
		return nil, errors.New(fmt.Sprint("context not found in supplied kubeconfig ", "contextName: ", contextName, " configFileLocation: ", path))
	}
	overrides := &clientcmd.ConfigOverrides{
		Context:     *context,
		ClusterInfo: *clientConfig.Clusters[context.Cluster],
		AuthInfo:    *clientConfig.AuthInfos[context.AuthInfo],
	}

	cConfig := clientcmd.NewNonInteractiveClientConfig(*clientConfig, contextName, overrides, clientcmd.NewDefaultClientConfigLoadingRules())
	restConfig, err = cConfig.ClientConfig()

	if err != nil {
		return nil, err
	}

	return client.New(restConfig, client.Options{})
}

func GetKubeconfigFileLocation(location string) (string, error) {
	if location != "" {
		return location, nil
	} else if kubeconfigEnvVar, found := os.LookupEnv("KUBECONFIG"); found {
		return kubeconfigEnvVar, nil
	} else {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(homeDir, ".kube", "config"), nil
	}
}

func KubeconfigToHost(configFileLocation string, contextName string, overrideSourceIP string, overrideSourcePort string) (string, error) {
	if overrideSourceIP != "" && overrideSourcePort != "" {
		return fmt.Sprintf("https://%s:%s", overrideSourceIP, overrideSourcePort), nil
	}
	path, err := GetKubeconfigFileLocation(configFileLocation)
	if err != nil {
		return "", err
	}

	clientConfig, err := clientcmd.LoadFromFile(path)

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
		return "", errors.New(fmt.Sprint("context not found in supplied kubeconfig ", "contextName: ", contextName, " configFileLocation: ", path))
	}
	return clientConfig.Clusters[context.Cluster].Server, nil
}
