// Copyright 2018 The MATRIX Authors 
// This file is part of the MATRIX library. 
// 
// The MATRIX library is free software: you can redistribute it and/or modify 
// it under the terms of the GNU Lesser General Public License as published by 
// the Free Software Foundation, either version 3 of the License, or 
// (at your option) any later version. 
// 
// The MATRIX library is distributed in the hope that it will be useful, 
// but WITHOUT ANY WARRANTY; without even the implied warranty of 
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the 
// GNU Lesser General Public License for more details. 
// 
// You should have received a copy of the GNU Lesser General Public License 
// along with the MATRIX library. If not, see <http://www.gnu.org/licenses/>. 
package manhash

import (
	"github.com/ethereum/go-ethereum/election"
	"github.com/ethereum/go-ethereum/log"
)

type Election struct {
	MinerNet     MinerNodeList
	CommitteeNet CommitteeNodeList
	org_data     map[string]election.NodeInfo
}

func (e *Election) GenNetwork(n *election.NodeList, networkGenerated chan<- bool) {
	log.Info("election msg: input node list", "nodeList", n)
	e.org_data = make(map[string]election.NodeInfo)
	for _, node := range n.MinerList {
		e.org_data[node.ID] = node
	}
	for _, node := range n.CommitteeList {
		e.org_data[node.ID] = node
	}
	for _, node := range n.Both {
		e.org_data[node.ID] = node
	}

	if hasNodesToMinner, nodesToMinner := e.CommitteeNet.GetOrgGroupData(n.CommitteeList, n.Both); hasNodesToMinner {
		n.MinerList = append(n.MinerList, nodesToMinner...)
	}
	e.MinerNet.GetOrgGroupData(n.MinerList)
	e.CommitteeNet.GetRefreshData(n.OfflineList)
	e.MinerNet.GetRefreshData(n.OfflineList)

	e.CommitteeNet.DivGroup()
	e.CommitteeNet.GenNetwork()
	e.CommitteeNet.RefreshNetwork()

	e.MinerNet.DivGroup()
	e.MinerNet.GenNetwork()
	e.MinerNet.RefreshNetwork()
	networkGenerated <- true
}

func (e *Election) GetSuperMiner() []election.NodeInfo {
	superMinerNodeList := make([]election.NodeInfo, len(e.MinerNet.networks))
	for i, tree := range e.MinerNet.networks {
		id := tree.treeNode[0].globalIdx
		superMinerNodeList[i] = e.org_data[id]
	}
	return superMinerNodeList
}

func (e *Election) GetSuperCommittee() []election.NodeInfo {
	superCommitteeNodeList := make([]election.NodeInfo, len(e.CommitteeNet.networks))
	for i, tree := range e.CommitteeNet.networks {
		id := tree.treeNode[0].globalIdx
		superCommitteeNodeList[i] = e.org_data[id]
	}
	return superCommitteeNodeList
}

func (e *Election) GetIP(ID string) string {
	for _, node := range e.CommitteeNet.orgNodeList {
		if node.ID == ID {
			return node.IP
		}
	}
	for _, node := range e.MinerNet.orgNodeList {
		if node.ID == ID {
			return node.IP
		}
	}
	return ""
}

func (e *Election) getTree(id string) (int, ManTree) {
	for _, tree := range e.CommitteeNet.networks {
		for _, node := range tree.treeNode {
			if node.globalIdx == id {
				return node.localIdx, tree
			}
		}
	}
	for _, tree := range e.MinerNet.networks {
		for _, node := range tree.treeNode {
			if node.globalIdx == id {
				return node.localIdx, tree
			}
		}
	}
	return -1, ManTree{}
}

func (e *Election) GetChild(id string) []election.NodeInfo {
	children := make([]election.NodeInfo, 0)
	selfLocalID, tree := e.getTree(id)
	if selfLocalID != -1 {
		for _, node := range tree.treeNode {
			if node.parentIdx == selfLocalID {
				children = append(children, e.org_data[node.globalIdx])
			}
		}
	}
	return children
}

func (e *Election) GetParent(id string) election.NodeInfo {
	selfLocalID, tree := e.getTree(id)
	if selfLocalID != -1 {
		for _, node := range tree.treeNode {
			if node.lChildIdx == selfLocalID || node.rChildIdx == selfLocalID {
				return e.org_data[node.globalIdx]
			}
		}
	}
	return election.NodeInfo{}
}

func (e *Election) GetLeafNode(id string) election.NodeInfo {
	children := e.GetChild(id)
	if len(children) == 0 {
		return e.org_data[id]
	} else {
		return e.GetLeafNode(children[0].ID) //always get from left child ?
	}
}

func (e *Election) GetIDType(id string) int {
	for _, tree := range e.CommitteeNet.networks {
		for i, node := range tree.treeNode {
			if node.globalIdx == id {
				if i == 0 {
					return 2
				} else {
					return 3
				}
			}
		}
	}
	for _, tree := range e.MinerNet.networks {
		for i, node := range tree.treeNode {
			if node.globalIdx == id {
				if i == 0 {
					return 0
				} else {
					return 1
				}
			}
		}
	}
	return -1
}
