package knative

import (
	"fmt"
	"time"

	clienteventingv1beta1 "knative.dev/client/pkg/eventing/v1beta1"
	clientservingv1 "knative.dev/client/pkg/serving/v1"
	eventingv1beta1 "knative.dev/eventing/pkg/client/clientset/versioned/typed/eventing/v1beta1"
	servingv1 "knative.dev/serving/pkg/client/clientset/versioned/typed/serving/v1"

	"github.com/boson-project/func/k8s"
)

const (
	DefaultWaitingTimeout = 60 * time.Second
)

func NewServingClient(namespace string) (clientservingv1.KnServingClient, error) {

	restConfig, err := k8s.GetClientConfig().ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create new serving client: %v", err)
	}

	servingClient, err := servingv1.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create new serving client: %v", err)
	}

	client := clientservingv1.NewKnServingClient(servingClient, namespace)

	return client, nil
}

func NewEventingClient(namespace string) (clienteventingv1beta1.KnEventingClient, error) {

	restConfig, err := k8s.GetClientConfig().ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create new serving client: %v", err)
	}

	eventingClient, err := eventingv1beta1.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create new eventing client: %v", err)
	}

	client := clienteventingv1beta1.NewKnEventingClient(eventingClient, namespace)

	return client, nil
}
