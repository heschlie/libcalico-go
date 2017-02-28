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

	ip := net.ParseIP(annotations["projectcalico.org/IPv4"])

	_, bgpV4Cidr, err := net.ParseCIDR(annotations["projectcalico.org/BGPIPv4Net"])
	if err != nil {
		log.Warnf("Could not parse BGPIPv4CIDR from k8s annotation: %s",
			annotations["projectcalico.org/BGPIPv4Net"])
	}

	bgpAsn, err := numorstring.ASNumberFromString(annotations["projectcalico.org/BGPASNumber"])
	if err != nil {
		log.Warnf("Could not parse ASNumber from k8s annotation: %s",
			annotations["projectcalico.org/BGPASNumber"])
	}

	log.Debugf("Node IP is %s", ip)
	if ip == nil {
		return nil, fmt.Errorf("Failed to parse IP '%s' received from k8s for Node", nodeIP)
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
	vals := kvp.Value.(model.Node)
	annotations := map[string]string {
		"projectcalico.org/BGPIPv4Addr": vals.BGPIPv4Addr.String(),
		"projectcalico.org/BGPASNumber": vals.BGPASNumber.String(),
		"projectcalico.org/BGPIPv4Net":  vals.BGPIPv4Net.String(),
	}

	nodeMeta := kapiv1.ObjectMeta{
		Name:        kvp.Key.(model.NodeKey).Hostname,
		Annotations: annotations,
		Labels:      vals.Labels,
	}

	node := &kapiv1.Node{
		ObjectMeta: nodeMeta,
	}

	return node, nil
}
