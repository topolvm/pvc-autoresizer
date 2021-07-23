package client

import (
	"context"
	"fmt"
	"strings"

	"github.com/topolvm/pvc-autoresizer/metrics"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	originalclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
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
	// https://github.com/kubernetes-sigs/controller-runtime/blob/v0.9.2/pkg/client/client.go#L257-L289
	var gvk schema.GroupVersionKind
	//lint:ignore S1034 ignore this
	switch list.(type) {
	case *unstructured.UnstructuredList:
		u, ok := list.(*unstructured.UnstructuredList)
		if !ok {
			return fmt.Errorf("unstructured client did not understand object: %T", list)
		}
		gvk = u.GroupVersionKind()
	case *metav1.PartialObjectMetadataList:
		gvk = list.GetObjectKind().GroupVersionKind()
	default:
		var err error
		gvk, err = apiutil.GVKForObject(list, c.Scheme())
		if err != nil {
			return err
		}
	}

	err := c.Client.List(ctx, list, opts...)
	if err != nil {
		metrics.KubernetesClientFailTotal.Increment(gvk.Group, gvk.Version, strings.TrimSuffix(gvk.Kind, "List"), "LIST")
	}
	return err
}

func (c *clientWrapper) Update(ctx context.Context, obj originalclient.Object, opts ...originalclient.UpdateOption) error {
	// https://github.com/kubernetes-sigs/controller-runtime/blob/v0.9.2/pkg/client/client.go#L304-L315
	var gvk schema.GroupVersionKind
	//lint:ignore S1034 ignore this
	switch obj.(type) {
	case *unstructured.Unstructured:
		u, ok := obj.(*unstructured.Unstructured)
		if !ok {
			return fmt.Errorf("unstructured client did not understand object: %T", obj)
		}
		gvk = u.GroupVersionKind()
	case *metav1.PartialObjectMetadata:
		return fmt.Errorf("cannot update using only metadata -- did you mean to patch?")
	default:
		var err error
		gvk, err = apiutil.GVKForObject(obj, c.Scheme())
		if err != nil {
			return err
		}
	}

	err := c.Client.Update(ctx, obj, opts...)
	if err != nil {
		metrics.KubernetesClientFailTotal.Increment(gvk.Group, gvk.Version, gvk.Kind, "PUT")
	}
	return err
}
