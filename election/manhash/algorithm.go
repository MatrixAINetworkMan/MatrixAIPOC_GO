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

const maxTreeNodeNum = 1024
const nullNodeIdx = -1
const maxMasterNodeNum = 1024 * 32

type treeNode struct {
	lChildIdx      int
	rChildIdx      int
	parentIdx      int
	localIdx       int
	globalIdx      string
	upGradeNodeIdx int
}

//ManTree
type ManTree struct {
	treeNode [maxTreeNodeNum]treeNode
	Depth    int
}

type listNodeToTreeNodeInfo struct {
	treeType      int //0:election network；1：committee network
	regionIndex   int //
	treeNodeIndex int //
}

//AddNode
func (tree *ManTree) AddNode(localIdx int, globalIdx string, parentIdx int, leftChild bool, state bool, level int) {
	tree.treeNode[localIdx].localIdx = localIdx
	tree.treeNode[localIdx].lChildIdx = nullNodeIdx
	tree.treeNode[localIdx].rChildIdx = nullNodeIdx
	tree.treeNode[localIdx].parentIdx = parentIdx
	tree.treeNode[localIdx].upGradeNodeIdx = localIdx
	tree.treeNode[localIdx].globalIdx = globalIdx

	if depth := level + 1; depth > tree.Depth {
		tree.Depth = depth
	}

	if parentIdx == nullNodeIdx {
		return
	}

	if leftChild {
		tree.treeNode[parentIdx].lChildIdx = localIdx
	} else {
		tree.treeNode[parentIdx].rChildIdx = localIdx
	}
}

func (tree *ManTree) Clear() {
	for i := 0; i < maxTreeNodeNum; i++ {
		tree.treeNode[i].localIdx = nullNodeIdx
		tree.treeNode[i].lChildIdx = nullNodeIdx
		tree.treeNode[i].rChildIdx = nullNodeIdx
		tree.treeNode[i].parentIdx = nullNodeIdx
		tree.treeNode[i].globalIdx = ""
		tree.treeNode[i].upGradeNodeIdx = nullNodeIdx
	}
	tree.Depth = 0
}

func (tree *ManTree) IsEmpty() bool {
	return tree.treeNode[0].localIdx == nullNodeIdx
}

func setListIdx2TreeIdxTbl(s *listNodeToTreeNodeInfo, regionIndex int, netFlag int, treeIndex int) {
	s.treeNodeIndex = treeIndex
	s.regionIndex = regionIndex
	s.treeType = netFlag
}

//
func (tree *ManTree) GenTree(list []string, regionIndex int, netFlag int, mapTblPtr *[maxMasterNodeNum]listNodeToTreeNodeInfo) {

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

	tree.AddNode(0, list[0], nullNodeIdx, true, true, 0)
	//setListIdx2TreeIdxTbl(&mapTblPtr[list[0]], regionIndex, netFlag, 0)

	if nodeLen <= 1 {
		return
	}

	parentStartIndex := 0
	parentLevelWidth := 1
	nodeCnt := 1
	for levelLoop := 1; levelLoop < maxLevelNum; levelLoop++ {
		//left
		for levelNodeLoop := 0; levelNodeLoop < parentLevelWidth; levelNodeLoop++ {
			tree.AddNode(nodeCnt, list[nodeCnt], parentStartIndex+levelNodeLoop, true, true, levelLoop)
			//setListIdx2TreeIdxTbl(&mapTblPtr[list[nodeCnt]], regionIndex, netFlag, nodeCnt)
			nodeCnt++
			if nodeCnt >= nodeLen {
				return
			}
		}
		//right
		for levelNodeLoop := 0; levelNodeLoop < parentLevelWidth; levelNodeLoop++ {
			tree.AddNode(nodeCnt, list[nodeCnt], parentStartIndex+levelNodeLoop, false, true, levelLoop)
			//setListIdx2TreeIdxTbl(&mapTblPtr[list[nodeCnt]], regionIndex, netFlag, nodeCnt)
			nodeCnt++
			if nodeCnt >= nodeLen {
				return
			}
		}

		parentStartIndex = parentStartIndex + parentLevelWidth
		parentLevelWidth = parentLevelWidth * 2
	}
}

func (tree *ManTree) GetSpecLevelNode(level int) []treeNode {
	if level < 0 {
		return nil
	}
	var start int = 0
	for i := 0; i < level; i++ {
		start = start + (1 << uint8(i))
	}
	levelLen := (1 << uint8(level))

	var levelNode []treeNode
	for i := start; i < start+levelLen; i++ {
		if tree.treeNode[i].localIdx != nullNodeIdx {
			levelNode = append(levelNode, tree.treeNode[i])
		}
	}
	if len(levelNode) == 0 {
		return nil
	}
	return levelNode
}

//
func preorderReplace(nodeIndex int, treeNode *[maxTreeNodeNum]treeNode) int {
	var srchIndex int
	if nodeIndex >= maxTreeNodeNum {
		return nullNodeIdx
	}
	if treeNode[nodeIndex].localIdx != nullNodeIdx {
		if treeNode[nodeIndex].localIdx == treeNode[nodeIndex].upGradeNodeIdx {
			return treeNode[nodeIndex].localIdx
		}
		if treeNode[nodeIndex].lChildIdx != nullNodeIdx {
			if srchIndex = preorderReplace(treeNode[nodeIndex].lChildIdx, treeNode); srchIndex != nullNodeIdx {
				return srchIndex
			}
		}
		if treeNode[nodeIndex].rChildIdx != nullNodeIdx {
			if srchIndex = preorderReplace(treeNode[nodeIndex].rChildIdx, treeNode); srchIndex != nullNodeIdx {
				return srchIndex
			}

		}
		return nullNodeIdx
	} else {
		return nullNodeIdx
	}
}

func (tree *ManTree) refresh(localIndex int) {
	//set upgradeNode to nil if offline
	tree.treeNode[localIndex].upGradeNodeIdx = nullNodeIdx

	if localIndex == 0 {
		return
	}

	//searching for the backup node
	if replaceIndex := preorderReplace(localIndex, &tree.treeNode); replaceIndex != nullNodeIdx {
		tree.treeNode[localIndex].upGradeNodeIdx = replaceIndex
	} else {
		tree.treeNode[localIndex].upGradeNodeIdx = nullNodeIdx
	}
	return
}
