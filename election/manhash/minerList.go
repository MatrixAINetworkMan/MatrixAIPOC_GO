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
	"math"
	"math/rand"
	"github.com/ethereum/go-ethereum/election"
	"sort"
)

const maxMinerGroupNum = 32

var minerListMapTreeTbl [maxMasterNodeNum]listNodeToTreeNodeInfo

type MinerNodeList struct {
	orgNodeList    []election.NodeInfo
	nodesToRefresh map[string]string
	nodeGroups     [][]election.NodeInfo
	networks       []ManTree
}

func (n *MinerNodeList) Less(i, j int) bool {
	if n.orgNodeList[i].Value == n.orgNodeList[j].Value {
		return n.orgNodeList[i].TxHash > n.orgNodeList[j].TxHash
	}
	return n.orgNodeList[i].Value > n.orgNodeList[j].Value
}

func (n *MinerNodeList) Swap(i, j int) {
	n.orgNodeList[i], n.orgNodeList[j] = n.orgNodeList[j], n.orgNodeList[i]
}

func (n *MinerNodeList) Len() int {
	return len(n.orgNodeList)
}

func caluNodeValue(n *election.NodeInfo, timeLevel float64) {
	//var coreTPS = 0.25 //4 * ori
	//var coreSTK = 0.2
	var coreTPS = 1 //4 * ori
	var coreSTK = 1 //5 * ori
	//
	tpsLevel := int(n.TPS/2000)
	if tpsLevel > 5 {
		tpsLevel = 5
	}
	//
	var wealthLevel int
	switch {
	case n.Wealth < 20000:
		wealthLevel = 100
	case n.Wealth < 40000:
		wealthLevel = 215
	default:
		wealthLevel = 450
	}
	n.Value = uint64((uint64(coreTPS)*uint64(tpsLevel) + uint64(coreSTK)*uint64(wealthLevel)) * uint64(timeLevel))
	//n.Value =0
}

func (n *MinerNodeList) GetOrgGroupData(nodes []election.NodeInfo) {
	//sort by online time first to calculate value
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].OnlineTime > nodes[j].OnlineTime
	})
	//calculate value
	nodesLength := len(nodes)
	var timeLevelInterval = float64(nodesLength) / 5.0
	for i := 0; i < nodesLength; i++ {
		var timeLevel = math.Pow(2, (float64(i)/timeLevelInterval)) * 0.25
		caluNodeValue(&nodes[i], timeLevel)
	}
	//copy
	n.orgNodeList = make([]election.NodeInfo, nodesLength)
	copy(n.orgNodeList, nodes)

	sort.Sort(n)
}

func (n *MinerNodeList) GetRefreshData(nodes []election.NodeInfo) {
	refreshNodeSize := len(nodes)
	if refreshNodeSize != 0 {
		n.nodesToRefresh = make(map[string]string)
		for _, node := range nodes {
			n.nodesToRefresh[node.ID] = node.ID
		}
	}
}

func (n *MinerNodeList) DivGroup() {
	nodesNum := len(n.orgNodeList)
	if nodesNum == 0 {
		return
	}

	nodesInEachGroup := nodesNum/maxMinerGroupNum + 1
	groupCount := maxMinerGroupNum
	if nodesInEachGroup == 1 {
		groupCount = nodesNum
	}
	n.nodeGroups = make([][]election.NodeInfo, groupCount)
	for i := range n.nodeGroups {
		n.nodeGroups[i] = make([]election.NodeInfo, nodesInEachGroup)
	}
	//divide groups
	for i := 0; i < nodesNum; i++ {
		n.nodeGroups[i%maxMinerGroupNum][i/maxMinerGroupNum] = n.orgNodeList[i]
		//
		//if n.orgNodeList[i].ID == self.ID { self.group = i%maxMinerGroupNum }
	}

	//pick super node
	for i := range n.nodeGroups {
		n.PickSuperNode(i)
	}
}

func (n *MinerNodeList) PickSuperNode(group int) {
	valueRange := make([]uint64, len(n.nodeGroups[group]))
	var sum uint64
	sum = 0
	for i := range n.nodeGroups[group] {
		sum += n.nodeGroups[group][i].Value
		valueRange[i] = sum
	}

	//var randSeed int64 = 1 // "to do"
	//rand.Seed(randSeed)
	if sum == 0 {
		return
	}
	randomValue := rand.Uint64() * sum
	for i, v := range valueRange {
		if v > randomValue {
			for j := i; j > 0; j-- {
				n.nodeGroups[group][j], n.nodeGroups[group][j-1] = n.nodeGroups[group][j-1], n.nodeGroups[group][j]
			}
			break
		}
	}
}

func (n *MinerNodeList) GenNetwork() {
	n.networks = make([]ManTree, len(n.nodeGroups))
	for i := range n.nodeGroups {
		n.networks[i].Clear()
		nodeNum := maxTreeNodeNum
		if len(n.nodeGroups[i]) < maxTreeNodeNum {
			nodeNum = len(n.nodeGroups[i])
		}
		IDList := make([]string, nodeNum)
		for j := range n.nodeGroups[i] {
			IDList[j] = n.nodeGroups[i][j].ID
		}
		n.networks[i].GenTree(IDList, 1, 1, &minerListMapTreeTbl)
	}
}

func (n *MinerNodeList) RefreshNetwork() {
	for _, net := range n.networks {
		for _, node := range net.treeNode {
			if _, ok := n.nodesToRefresh[node.globalIdx]; ok {
				net.refresh(node.localIdx)
			}
		}
	}
}
