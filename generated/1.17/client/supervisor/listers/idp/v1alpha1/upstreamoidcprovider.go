// Copyright 2020 the Pinniped contributors. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Code generated by lister-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "go.pinniped.dev/generated/1.17/apis/supervisor/idp/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// UpstreamOIDCProviderLister helps list UpstreamOIDCProviders.
type UpstreamOIDCProviderLister interface {
	// List lists all UpstreamOIDCProviders in the indexer.
	List(selector labels.Selector) (ret []*v1alpha1.UpstreamOIDCProvider, err error)
	// UpstreamOIDCProviders returns an object that can list and get UpstreamOIDCProviders.
	UpstreamOIDCProviders(namespace string) UpstreamOIDCProviderNamespaceLister
	UpstreamOIDCProviderListerExpansion
}

// upstreamOIDCProviderLister implements the UpstreamOIDCProviderLister interface.
type upstreamOIDCProviderLister struct {
	indexer cache.Indexer
}

// NewUpstreamOIDCProviderLister returns a new UpstreamOIDCProviderLister.
func NewUpstreamOIDCProviderLister(indexer cache.Indexer) UpstreamOIDCProviderLister {
	return &upstreamOIDCProviderLister{indexer: indexer}
}

// List lists all UpstreamOIDCProviders in the indexer.
func (s *upstreamOIDCProviderLister) List(selector labels.Selector) (ret []*v1alpha1.UpstreamOIDCProvider, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.UpstreamOIDCProvider))
	})
	return ret, err
}

// UpstreamOIDCProviders returns an object that can list and get UpstreamOIDCProviders.
func (s *upstreamOIDCProviderLister) UpstreamOIDCProviders(namespace string) UpstreamOIDCProviderNamespaceLister {
	return upstreamOIDCProviderNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// UpstreamOIDCProviderNamespaceLister helps list and get UpstreamOIDCProviders.
type UpstreamOIDCProviderNamespaceLister interface {
	// List lists all UpstreamOIDCProviders in the indexer for a given namespace.
	List(selector labels.Selector) (ret []*v1alpha1.UpstreamOIDCProvider, err error)
	// Get retrieves the UpstreamOIDCProvider from the indexer for a given namespace and name.
	Get(name string) (*v1alpha1.UpstreamOIDCProvider, error)
	UpstreamOIDCProviderNamespaceListerExpansion
}

// upstreamOIDCProviderNamespaceLister implements the UpstreamOIDCProviderNamespaceLister
// interface.
type upstreamOIDCProviderNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all UpstreamOIDCProviders in the indexer for a given namespace.
func (s upstreamOIDCProviderNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.UpstreamOIDCProvider, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.UpstreamOIDCProvider))
	})
	return ret, err
}

// Get retrieves the UpstreamOIDCProvider from the indexer for a given namespace and name.
func (s upstreamOIDCProviderNamespaceLister) Get(name string) (*v1alpha1.UpstreamOIDCProvider, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("upstreamoidcprovider"), name)
	}
	return obj.(*v1alpha1.UpstreamOIDCProvider), nil
}
