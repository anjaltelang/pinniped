/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package controller

// ErrSyntheticRequeue can be returned from a Syncer to force a retry artificially for the current key.
// This can also be done by re-adding the key to queue, but this is more convenient and has better logging.
const ErrSyntheticRequeue = constErr("synthetic requeue request")

var _ error = constErr("")

type constErr string

func (e constErr) Error() string {
	return string(e)
}
