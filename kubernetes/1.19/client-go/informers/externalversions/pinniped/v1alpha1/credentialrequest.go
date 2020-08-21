/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Code generated by informer-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	time "time"

	pinnipedv1alpha1 "github.com/suzerain-io/pinniped/kubernetes/1.19/api/apis/pinniped/v1alpha1"
	versioned "github.com/suzerain-io/pinniped/kubernetes/1.19/client-go/clientset/versioned"
	internalinterfaces "github.com/suzerain-io/pinniped/kubernetes/1.19/client-go/informers/externalversions/internalinterfaces"
	v1alpha1 "github.com/suzerain-io/pinniped/kubernetes/1.19/client-go/listers/pinniped/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// CredentialRequestInformer provides access to a shared informer and lister for
// CredentialRequests.
type CredentialRequestInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.CredentialRequestLister
}

type credentialRequestInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// NewCredentialRequestInformer constructs a new informer for CredentialRequest type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewCredentialRequestInformer(client versioned.Interface, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredCredentialRequestInformer(client, resyncPeriod, indexers, nil)
}

// NewFilteredCredentialRequestInformer constructs a new informer for CredentialRequest type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredCredentialRequestInformer(client versioned.Interface, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.PinnipedV1alpha1().CredentialRequests().List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.PinnipedV1alpha1().CredentialRequests().Watch(context.TODO(), options)
			},
		},
		&pinnipedv1alpha1.CredentialRequest{},
		resyncPeriod,
		indexers,
	)
}

func (f *credentialRequestInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredCredentialRequestInformer(client, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *credentialRequestInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&pinnipedv1alpha1.CredentialRequest{}, f.defaultInformer)
}

func (f *credentialRequestInformer) Lister() v1alpha1.CredentialRequestLister {
	return v1alpha1.NewCredentialRequestLister(f.Informer().GetIndexer())
}
