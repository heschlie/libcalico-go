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
	"github.com/projectcalico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/libcalico-go/lib/net"
	"github.com/projectcalico/libcalico-go/lib/numorstring"
	kapiv1 "k8s.io/client-go/pkg/api/v1"
	"fmt"
)

// Convert a Kubernetes format node, with Calico annotations, to a Calico Node
func K8sNodeToCalico(node *kapiv1.Node) (*model.KVPair, error) {
	kvp := model.KVPair{
		Key: model.NodeKey{
			Hostname: node.Name,
		},
		Revision: node.ObjectMeta.ResourceVersion,
	}

	calicoNode := model.Node{}

	annotations := node.ObjectMeta.Annotations

	cidrString, ok := annotations["projectcalico.org/IPv4Address"]; if ok {
		ip, cidr, err := net.ParseCIDR(cidrString)
		if err != nil {
			log.Errorf("Could not parse BGPIPv4CIDR from k8s annotation: %s", cidrString)
			return nil, err
		}

		calicoNode.FelixIPv4   = ip
		calicoNode.BGPIPv4Addr = ip
		calicoNode.BGPIPv4Net  = cidr
	}

	asnString, ok := annotations["projectcalico.org/ASNumber"]; if ok {
		asn, err := numorstring.ASNumberFromString(asnString)
		if err != nil {
			log.Errorf("Could not parse ASNumber from k8s annotation: %s", asnString)
			return nil, err
		}

		calicoNode.BGPASNumber = &asn
	}

	kvp.Value = &calicoNode

	return &kvp, nil
}

// We take a k8s node and a Calico node and push the values from the Calico node into the k8s node
func MakeK8sNode(kvp *model.KVPair, node *kapiv1.Node) (*kapiv1.Node, error) {
	calicoNode := kvp.Value.(*model.Node)
	log.Debugf("Converting to k8s Node:\n%+v", calicoNode)

	// In order to make sure we always end up with a CIDR that has the IP and not just network
	// we assemble the CIDR from BGPIPv4 and FelixIPv4.
	subnet := calicoNode.BGPIPv4Net.Mask.String()
	log.Debug(subnet)
	log.Debug(calicoNode.BGPIPv4Addr.String())
	ipCidr := fmt.Sprintf("%s/%s", calicoNode.BGPIPv4Addr.String(), subnet)
	node.Annotations["projectcalico.org/IPv4Address"] = ipCidr

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
