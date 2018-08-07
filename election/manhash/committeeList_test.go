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
