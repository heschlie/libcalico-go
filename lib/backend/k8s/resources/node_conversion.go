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
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/libcalico-go/lib/net"
	"github.com/projectcalico/libcalico-go/lib/numorstring"
	kapiv1 "k8s.io/client-go/pkg/api/v1"
)

// Convert a Kubernetes format node, with Calico annotations, to a Calico Node
func K8sNodeToCalico(node *kapiv1.Node) (*model.KVPair, error) {
	annotations := node.ObjectMeta.Annotations

	ip := net.ParseIP(annotations["projectcalico.org/IPv4Address"])
	log.Debugf("Node IP is %s", ip)
	if ip == nil {
		return nil, errors.New("Invalid IP received from k8s for Node")
	}

	var err error
	var bgpV4Cidr *net.IPNet
	cidrString, ok := annotations["projectcalico.org/BGPIPv4Net"]; if ok {
		_, bgpV4Cidr, err = net.ParseCIDR(cidrString)
		if err != nil {
			log.Errorf("Could not parse BGPIPv4CIDR from k8s annotation: %s", cidrString)
		}
	}

	var bgpAsn numorstring.ASNumber
	asnString, ok := annotations["projectcalico.org/ASNumber"]; if ok {
		bgpAsn, err = numorstring.ASNumberFromString(asnString)
		if err != nil {
			log.Errorf("Could not parse ASNumber from k8s annotation: %s", asnString)
		}
	}

	// We want bgpAsn to be nil if it is not set, and not 0
	if bgpAsn == 0 {
		bgpAsn = nil
	}

	return &model.KVPair{
		Key: model.NodeKey{
			Hostname: node.Name,
		},
		Value: &model.Node{
			FelixIPv4:   ip,
			Labels:      node.Labels,
			BGPIPv4Addr: ip,
			BGPASNumber: &bgpAsn,
			BGPIPv4Net:  bgpV4Cidr,
		},
		Revision: node.ObjectMeta.ResourceVersion,
	}, nil
}

// Convert a Calico Node to Kubernetes, place BGP configuration info in annotations
func CalicoToK8sNode(kvp *model.KVPair) (*kapiv1.Node, error) {
	calicoNode := kvp.Value.(model.Node)
	annotations := map[string]string {
		"projectcalico.org/IPv4Address": calicoNode.BGPIPv4Addr.String(),
		"projectcalico.org/ASNumber":    calicoNode.BGPASNumber.String(),
		"projectcalico.org/BGPIPv4Net":  calicoNode.BGPIPv4Net.String(),
	}

	nodeMeta := kapiv1.ObjectMeta{
		Name:        kvp.Key.(model.NodeKey).Hostname,
		Annotations: annotations,
		Labels:      calicoNode.Labels,
	}

	node := &kapiv1.Node{
		ObjectMeta: nodeMeta,
	}

	return node, nil
}

// We take a k8s node and a Calico node and push the values from the Calico node into the k8s node
func MakeK8sNode(kvp *model.KVPair, node *kapiv1.Node) (*kapiv1.Node, error) {
	calicoNode := kvp.Value.(model.Node)
	node.Annotations["projectcalico.org/IPv4Address"] = calicoNode.BGPIPv4Addr.String()
	node.Annotations["projectcalico.org/BGPIPv4Net"] = calicoNode.BGPIPv4Net.String()

	// Don't set the ASNumber if it is nil
	if calicoNode.BGPASNumber != nil {
		node.Annotations["projectcalico.org/ASNumber"] = calicoNode.BGPASNumber.String()
	}

	// Add all of the calicoNode labels into the k8s node
	for k, v := range calicoNode.Labels {
		node.Labels[k] = v
	}

	return node, nil
}
