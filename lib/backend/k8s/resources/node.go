// Copyright (c) 2016 Tigera, Inc. All rights reserved.

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package resources

import (
	log "github.com/Sirupsen/logrus"

	"github.com/projectcalico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/libcalico-go/lib/errors"

	"k8s.io/client-go/kubernetes"
	kapiv1 "k8s.io/client-go/pkg/api/v1"
	metav1 "k8s.io/client-go/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

func NewNodeClient(c *kubernetes.Clientset, r *rest.RESTClient) api.Client {
	return &nodeClient{
		clientSet: c,
	}
}

// Implements the api.Client interface for Nodes.
type nodeClient struct {
	clientSet *kubernetes.Clientset
}

func (c *nodeClient) Create(kvp *model.KVPair) (*model.KVPair, error) {
	log.Warn("Operation Create is not supported on Node type")
	return nil, errors.ErrorOperationNotSupported{
		Identifier: kvp.Key,
		Operation:  "Create",
	}
}

func (c *nodeClient) Update(kvp *model.KVPair) (*model.KVPair, error) {
	// Get a current copy of the node to fill in fields we don't track
	oldNode, err := c.clientSet.Nodes().Get(kvp.Key.(model.NodeKey).Hostname, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	node, err := MakeK8sNode(kvp, oldNode)
	if err != nil {
		return nil, err
	}

	newNode, err := c.clientSet.Nodes().Update(node)
	if err != nil {
		return nil, K8sErrorToCalico(err, kvp.Key)
	}

	newCalicoNode, err := K8sNodeToCalico(newNode)
	if err != nil {
		log.Errorf("Failed to parse returned Node after call to update %+v", newNode)
		return nil, err
	}

	return newCalicoNode, nil
}

func (c *nodeClient) Apply(kvp *model.KVPair) (*model.KVPair, error) {
	node, err := c.Update(kvp)
	if err != nil {
		if _, ok := err.(errors.ErrorResourceDoesNotExist); !ok {
			return nil, err
		}
		log.Warnf("Node Apply could not find %s in k8s datastore", kvp.Key.(model.NodeKey).Hostname)

		// Create is not currently implemented, and probably will not be, but will throw an appropriate error
		// for the user, along with the above warning
		return c.Create(kvp)
	}
	return node, nil
}

func (c *nodeClient) Delete(kvp *model.KVPair) error {
	log.Warn("Operation Delete is not supported on Node type")
	return errors.ErrorOperationNotSupported{
		Identifier: kvp.Key,
		Operation:  "Delete",
	}
}

func (c *nodeClient) Get(key model.Key) (*model.KVPair, error) {
	log.Debug("Received Get request on Node type")
	node, err := c.clientSet.Nodes().Get(key.(model.NodeKey).Hostname, metav1.GetOptions{})
	if err != nil {
		return nil, K8sErrorToCalico(err, key)
	}

	kvp, err := K8sNodeToCalico(node)
	if err != nil {
		log.Panicf("%s", err)
	}

	return kvp, nil
}

func (c *nodeClient) List(list model.ListInterface) ([]*model.KVPair, error) {
	log.Debug("Received List request on Node type")
	nodes, err := c.clientSet.Nodes().List(kapiv1.ListOptions{})
	if err != nil {
		K8sErrorToCalico(err, list)
	}

	kvps := []*model.KVPair{}

	for _, node := range nodes.Items {
		n, err := K8sNodeToCalico(&node)
		if err != nil {
			log.Panicf("%s", err)
		}
		kvps = append(kvps, n)
	}

	return kvps, nil
}

func (c *nodeClient) EnsureInitialized() error {
	return nil
}

func (c *nodeClient) EnsureCalicoNodeInitialized(node string) error {
	return nil
}

func (c *nodeClient) Syncer(callbacks api.SyncerCallbacks) api.Syncer {
	return nodeFakeSyncer{}
}

type nodeFakeSyncer struct{}

func (f nodeFakeSyncer) Start() {}
