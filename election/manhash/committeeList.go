package manhash

import (
	"math"
	"math/rand"
	"github.com/ethereum/go-ethereum/election"
	"sort"
)

const maxCommitteeGroupLevel = 3
const committeeGroupNum = 11
const maxCommitteeNodes = 165

var listMapTreeTbl [maxMasterNodeNum]listNodeToTreeNodeInfo

type CommitteeNodeList struct {
	orgNodeList    []election.NodeInfo
	nodesToRefresh map[string]string
	nodeGroups     [][]election.NodeInfo
	networks       []ManTree
}

func (n *CommitteeNodeList) Less(i, j int) bool {
	if n.orgNodeList[i].Value == n.orgNodeList[j].Value {
		return n.orgNodeList[i].TxHash > n.orgNodeList[j].TxHash
	}
	return n.orgNodeList[i].Value > n.orgNodeList[j].Value
}

func (n *CommitteeNodeList) Swap(i, j int) {
	n.orgNodeList[i], n.orgNodeList[j] = n.orgNodeList[j], n.orgNodeList[i]
}

func (n *CommitteeNodeList) Len() int {
	return len(n.orgNodeList)
}

func (n *CommitteeNodeList) GetOrgGroupData(nodesForCommittee []election.NodeInfo, nodesForBoth []election.NodeInfo) (bool, []election.NodeInfo) {
	//merge the nodes
	nodes := make([]election.NodeInfo, len(nodesForCommittee)+len(nodesForBoth))
	copy(nodes, nodesForCommittee[:])
	copy(nodes[len(nodesForCommittee):], nodesForBoth[:])

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
	if len(n.orgNodeList) > maxCommitteeNodes {
		n.orgNodeList = n.orgNodeList[:maxCommitteeNodes]
		set := make(map[string]election.NodeInfo)
		for _, v := range n.orgNodeList {
			set[v.ID] = v
		}

		hasNodeLeft := false
		nodesLeft := make([]election.NodeInfo, 0)
		for _, v := range nodesForBoth {
			if _, ok := set[v.ID]; ok {
				continue
			}
			hasNodeLeft = true
			nodesLeft = append(nodesLeft, v)
		}
		return hasNodeLeft, nodesLeft
	}
	return false, nil
}

func (n *CommitteeNodeList) GetRefreshData(nodes []election.NodeInfo) {
	nodesToRefreshSize := len(nodes)
	if nodesToRefreshSize != 0 {
		n.nodesToRefresh = make(map[string]string)
		for _, node := range nodes {
			n.nodesToRefresh[node.ID] = node.ID
		}
	}
}

func (n *CommitteeNodeList) DivGroup() {
	nodesNum := len(n.orgNodeList)
	if nodesNum == 0 {
		return
	}

	nodesInEachGroup := nodesNum/committeeGroupNum + 1
	groupCount := committeeGroupNum
	if nodesInEachGroup == 1 {
		groupCount = nodesNum
	}
	n.nodeGroups = make([][]election.NodeInfo, groupCount)
	for i := range n.nodeGroups {
		n.nodeGroups[i] = make([]election.NodeInfo, nodesInEachGroup)
	}
	//divide groups
	for i := 0; i < nodesNum; i++ {
		n.nodeGroups[i%committeeGroupNum][i/committeeGroupNum] = n.orgNodeList[i]
		//
		//if n.orgNodeList[i].ID == self.ID { self.group = i%committeeGroupNum }
	}

	//pick super node
	for i := range n.nodeGroups {
		n.PickSuperNode(i)
	}
}

func (n *CommitteeNodeList) PickSuperNode(group int) {
	valueRange := make([]uint64, len(n.nodeGroups[group]))
	var  sum uint64
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

func (n *CommitteeNodeList) GenNetwork() {
	n.networks = make([]ManTree, len(n.nodeGroups))
	for i := range n.nodeGroups {
		n.networks[i].Clear()
		nodeNum := maxCommitteeNodes
		if len(n.nodeGroups[i]) < maxTreeNodeNum {
			nodeNum = len(n.nodeGroups[i])
		}
		IDList := make([]string, nodeNum)
		for j := range n.nodeGroups[i] {
			IDList[j] = n.nodeGroups[i][j].ID
		}
		n.networks[i].GenTree(IDList, 1, 1, &listMapTreeTbl)
	}
}

func (n *CommitteeNodeList) RefreshNetwork() {
	for _, net := range n.networks {
		for _, node := range net.treeNode {
			if _, ok := n.nodesToRefresh[node.globalIdx]; ok {
				net.refresh(node.localIdx)
			}
		}
	}
}
