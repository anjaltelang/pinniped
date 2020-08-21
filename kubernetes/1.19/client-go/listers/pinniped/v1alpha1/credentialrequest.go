/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Code generated by lister-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "github.com/suzerain-io/pinniped/kubernetes/1.19/api/apis/pinniped/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// CredentialRequestLister helps list CredentialRequests.
// All objects returned here must be treated as read-only.
type CredentialRequestLister interface {
	// List lists all CredentialRequests in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.CredentialRequest, err error)
	// Get retrieves the CredentialRequest from the index for a given name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1alpha1.CredentialRequest, error)
	CredentialRequestListerExpansion
}

// credentialRequestLister implements the CredentialRequestLister interface.
type credentialRequestLister struct {
	indexer cache.Indexer
}

// NewCredentialRequestLister returns a new CredentialRequestLister.
func NewCredentialRequestLister(indexer cache.Indexer) CredentialRequestLister {
	return &credentialRequestLister{indexer: indexer}
}

// List lists all CredentialRequests in the indexer.
func (s *credentialRequestLister) List(selector labels.Selector) (ret []*v1alpha1.CredentialRequest, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.CredentialRequest))
	})
	return ret, err
}

// Get retrieves the CredentialRequest from the index for a given name.
func (s *credentialRequestLister) Get(name string) (*v1alpha1.CredentialRequest, error) {
	obj, exists, err := s.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("credentialrequest"), name)
	}
	return obj.(*v1alpha1.CredentialRequest), nil
}
