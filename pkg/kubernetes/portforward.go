package kubernetes

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"github.com/k8ssandra/cass-operator/pkg/httphelper"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

func PortForwardMgmtPort(config *rest.Config, pod *corev1.Pod) (string, error) {
	_, targetPort, err := httphelper.BuildPodHostFromPod(pod)
	if err != nil {
		return "", err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return "", err
	}

	restClient := clientset.CoreV1().RESTClient()
	req := restClient.Post().
		Resource("pods").
		Namespace(pod.Namespace).
		Name(pod.Name).
		SubResource("portforward")

	transport, upgrader, err := spdy.RoundTripperFor(config)
	if err != nil {
		return "", err
	}

	stopChan := make(chan struct{}, 1)
	readyChan := make(chan struct{})
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)

	// Use a random local port and connect to the mgmt-api port
	ports := []string{fmt.Sprintf("0:%d", targetPort)}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", req.URL())

	fw, err := portforward.New(dialer, ports, stopChan, readyChan, out, errOut)
	if err != nil {
		return "", err
	}

	go func() {
		if err := fw.ForwardPorts(); err != nil {
			fmt.Printf("ForwardPorts error: %v\n", err)
		}
	}()

	select {
	case <-readyChan:
		localPort := extractLocalPort(out.String())
		return fmt.Sprintf("localhost:%s", localPort), nil
	case <-time.After(10 * time.Second):
		return "", fmt.Errorf("timeout waiting for port forwarding to be ready")
	}
}

func extractLocalPort(output string) string {
	// Example output: "Forwarding from 127.0.0.1:random_port -> 8080"
	// Extract the random_port from the output
	var localPort string
	fmt.Sscanf(output, "Forwarding from 127.0.0.1:%s -> 8080", &localPort)
	return localPort
}
