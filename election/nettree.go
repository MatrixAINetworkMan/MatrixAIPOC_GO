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
package election

import ()

const MaxTreeNodeNum = 512
const NullNodeIdx = -1

type TreeNode struct {
	lChildIdx      int //index of left children node in binary tree
	rChildIdx      int //index of right children node in binary tree
	parentIdx      int //index of parent nodes in binary tree
	localIdx       int //index of local node in binary tree
	globalIdx      int //index of local node in global masternode list
	upGradeNodeIdx int //status of local node
}

type ElectionTree struct {
	treeNode [MaxTreeNodeNum]TreeNode
	Depth    int
}

//add nodes in the tree, and update the tree depth
func (tree *ElectionTree) AddNode(localIdx int, globalIdx int, parentIdx int, leftChild bool, state bool, level int) {
	tree.treeNode[localIdx].localIdx = localIdx
	tree.treeNode[localIdx].lChildIdx = NullNodeIdx
	tree.treeNode[localIdx].rChildIdx = NullNodeIdx
	tree.treeNode[localIdx].parentIdx = parentIdx
	tree.treeNode[localIdx].upGradeNodeIdx = localIdx
	tree.treeNode[localIdx].globalIdx = globalIdx

	if depth := level + 1; depth > tree.Depth {
		tree.Depth = depth
	}

	if parentIdx == NullNodeIdx {
		return
	}

	if leftChild {
		tree.treeNode[parentIdx].lChildIdx = localIdx
	} else {
		tree.treeNode[parentIdx].rChildIdx = localIdx
	}

	return
}

func (tree *ElectionTree) Clear() {
	for i := 0; i < MaxTreeNodeNum; i++ {
		tree.treeNode[i].localIdx = NullNodeIdx
		tree.treeNode[i].lChildIdx = NullNodeIdx
		tree.treeNode[i].rChildIdx = NullNodeIdx
		tree.treeNode[i].globalIdx = NullNodeIdx
		tree.treeNode[i].globalIdx = NullNodeIdx
	}
	tree.Depth = 0
}

/*
  Func: Judge if the election tree is blank
  based on: if the initial node local index is blank
*/
func (tree *ElectionTree) IsEmpty() bool {
	return tree.treeNode[0].localIdx == NullNodeIdx
}

func setListIdx2TreeIdxTbl(s *listNodeToTreeNodeInfo, regionIndex int, netFlag int, treeIndex int) {
	s.treeNodeIndex = treeIndex
	s.regionIndex = regionIndex
	s.treeType = netFlag
}

//generate the tree structure based on the sequenced list format
//sequence format: level0-----level1---------level2--------
//In the same layer: left node----right node
func (tree *ElectionTree) GenTree(list []int, regionIndex int, netFlag int, mapTblPtr *[MaxMasterNodeNum]listNodeToTreeNodeInfo) {

	var nodeLen int

	if nodeLen = len(list); nodeLen < 1 {
		return
	}

	maxLevelNum := 1
	for {
		if nodeLen <= (1 << (uint8(maxLevelNum) - 1)) {
			break
		}
		maxLevelNum++
	}

	//root node first
	tree.AddNode(0, list[0], NullNodeIdx, true, true, 0)
	setListIdx2TreeIdxTbl(&mapTblPtr[list[0]], regionIndex, netFlag, 0)

	if nodeLen <= 1 {
		return
	}

	parentStartIndex := 0
	parentLevelWidth := 1
	nodeCnt := 1
	for levelLoop := 1; levelLoop < maxLevelNum; levelLoop++ {
		//then left node
		for levelNodeLoop := 0; levelNodeLoop < parentLevelWidth; levelNodeLoop++ {
			tree.AddNode(nodeCnt, list[nodeCnt], parentStartIndex+levelNodeLoop, true, true, levelLoop)
			setListIdx2TreeIdxTbl(&mapTblPtr[list[nodeCnt]], regionIndex, netFlag, nodeCnt)
			nodeCnt++
			if nodeCnt >= nodeLen {
				return
			}
		}
		//then right node
		for levelNodeLoop := 0; levelNodeLoop < parentLevelWidth; levelNodeLoop++ {
			tree.AddNode(nodeCnt, list[nodeCnt], parentStartIndex+levelNodeLoop, false, true, levelLoop)
			setListIdx2TreeIdxTbl(&mapTblPtr[list[nodeCnt]], regionIndex, netFlag, nodeCnt)
			nodeCnt++
			if nodeCnt >= nodeLen {
				return
			}
		}

		parentStartIndex = parentStartIndex + parentLevelWidth
		parentLevelWidth = parentLevelWidth * 2
	}
}
func (node TreeNode) GetSelfIP(electInfo *ElectionNetInfo) string {
	if node.localIdx == NullNodeIdx {
		return "xxx.xxx.xxx.xxx"
	}
	return electInfo.MasterList.masterNodeList[node.globalIdx].Ip
}
func (node TreeNode) GetParentIP(electInfo *ElectionNetInfo, tree ElectionTree) string {

	if node.parentIdx == NullNodeIdx {
		return "xxx.xxx.xxx.xxx"
	}

	return electInfo.MasterList.masterNodeList[tree.treeNode[node.parentIdx].globalIdx].Ip
}
func (tree *ElectionTree) GetSpecLevelNode(level int) []TreeNode {
	if level < 0 {
		return nil
	}
	var start int = 0
	for i := 0; i < level; i++ {
		start = start + (1 << uint8(i))
	}
	levelLen := (1 << uint8(level))

	var levelNode []TreeNode
	for i := start; i < start+levelLen; i++ {
		if tree.treeNode[i].localIdx != NullNodeIdx {
			levelNode = append(levelNode, tree.treeNode[i])
		}
	}
	if len(levelNode) == 0 {
		return nil
	}
	return levelNode
}

//preorder traversal
func preorderReplace(nodeIndex int, treeNode *[MaxTreeNodeNum]TreeNode) int {
	var srchIndex int
	if treeNode[nodeIndex].localIdx != NullNodeIdx {
		if treeNode[nodeIndex].localIdx == treeNode[nodeIndex].upGradeNodeIdx {
			return treeNode[nodeIndex].localIdx
		}
		if treeNode[nodeIndex].lChildIdx != NullNodeIdx {
			if srchIndex = preorderReplace(treeNode[nodeIndex].lChildIdx, treeNode); srchIndex != NullNodeIdx {
				return srchIndex
			}
		}
		if treeNode[nodeIndex].rChildIdx != NullNodeIdx {
			if srchIndex = preorderReplace(treeNode[nodeIndex].rChildIdx, treeNode); srchIndex != NullNodeIdx {
				return srchIndex
			}

		}
		return NullNodeIdx
	} else {
		return NullNodeIdx
	}
}

func (tree *ElectionTree) refresh(localIndex int) {
	//when the local node goes offline, set the upgradenode to blank first
	tree.treeNode[localIndex].upGradeNodeIdx = NullNodeIdx

	//If the local node is the root node, no processing on upgrade backup node
	if localIndex == 0 {
		return
	}

	//Use preorder traversal to search the child tree for backup node
	if replaceIndex := preorderReplace(localIndex, &tree.treeNode); replaceIndex != NullNodeIdx {
		tree.treeNode[localIndex].upGradeNodeIdx = replaceIndex
	} else {
		tree.treeNode[localIndex].upGradeNodeIdx = NullNodeIdx
	}
	return
}
