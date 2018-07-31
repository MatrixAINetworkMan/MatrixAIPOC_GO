package manhash

import (
	"fmt"
	"matrix/election"
	"testing"
)

func TestDiv(t *testing.T) {
	var test struct {
		sample CommitteeNodeList
		expect CommitteeNodeList
	}
	//gen sample data
	sampleCommitteeData := make([]election.NodeInfo, 85)
	sampleBothData := make([]election.NodeInfo, 85)
	for i := range sampleCommitteeData {
		sampleCommitteeData[i] = election.NodeInfo{
			TPS:        uint32(i),
			IP:         "",
			ID:         string(i),
			Wealth:     uint64(i),
			OnlineTime: uint64(i),
			TxHash:     uint64(i),
			Value:      0,
		}
		sampleBothData[i] = election.NodeInfo{
			TPS:        uint32(i),
			IP:         "",
			ID:         string(i + 85),
			Wealth:     uint64(i),
			OnlineTime: uint64(i),
			TxHash:     uint64(i),
			Value:      0,
		}
	}

	test.expect.orgNodeList = make([]election.NodeInfo, len(sampleCommitteeData))
	copy(test.expect.orgNodeList, sampleCommitteeData)

	var hasNodeLeft bool
	var leftNodes []election.NodeInfo
	hasNodeLeft, leftNodes = test.sample.GetOrgGroupData(sampleCommitteeData, sampleBothData)
	if hasNodeLeft {
		fmt.Println(leftNodes)
	}
	test.sample.DivGroup()
	test.sample.GenNetwork()

	for _, net := range test.sample.networks {
		fmt.Println("treeDepth", net.Depth)
		n := 0
		for _, node := range net.treeNode {
			if node.localIdx == -1 {
				break
			}
			fmt.Println(node)
			n++
		}
	}

	//fmt.Println(test.sample.nodeGroups)
}
