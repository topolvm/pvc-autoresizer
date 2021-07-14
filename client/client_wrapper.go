package client

import (
	"context"
	"strings"

	"github.com/topolvm/pvc-autoresizer/metrics"
	originalclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type clientWrapper struct {
	originalclient.Client
}

func NewClientWrapper(c originalclient.Client) originalclient.Client {
	return &clientWrapper{
		Client: c,
	}
}

func (c *clientWrapper) List(ctx context.Context, list originalclient.ObjectList, opts ...originalclient.ListOption) error {
	gvk := list.GetObjectKind().GroupVersionKind()
	err := c.Client.List(ctx, list, opts...)
	if err != nil {
		metrics.KubernetesClientFailTotal.Increment(gvk.Group, gvk.Version, strings.TrimSuffix(gvk.Kind, "List"), "LIST")
	}
	return err
}

func (c *clientWrapper) Update(ctx context.Context, obj originalclient.Object, opts ...originalclient.UpdateOption) error {
	gvk := obj.GetObjectKind().GroupVersionKind()
	err := c.Client.Update(ctx, obj, opts...)
	if err != nil {
		metrics.KubernetesClientFailTotal.Increment(gvk.Group, gvk.Version, gvk.Kind, "PUT")
	}
	return err
}
