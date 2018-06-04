// Copyright 2018 The Matrix Authors
// This file is part of the Matrix library.
//
// The Matrix library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The Matrix library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the Matrix library. If not, see <http://www.gnu.org/licenses/>.

package election

import (
	//"github.com/ethereum/go-ethereum/core/types"

	//"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/log"

	"math/rand"
	"sort"
)

const (
	MaxCalNetTreeNum         = 2
	MaxVerifyNetTreeNum      = 1
	MaxTreeNum               = MaxCalNetTreeNum + MaxVerifyNetTreeNum
	MaxMasterNodeNum         = 10000
	CalcullatorNetFlag       = 0
	VerifierNetFlag          = 1

)

var LocalMasterNodeInfo = ListNodeInfo{
AnnonceRate:     1000,
Ip:              "192.168.123.72",
MortgageAccount: 10000,
Upime:           1000,
TxHash:          10000,
}



type listNodeToTreeNodeInfo struct {
	treeType      int //0: election network; 1: verification network
	regionIndex   int //group index
	treeNodeIndex int //storage location in the tree
}

type ElectionNetInfo struct {
	MasterList           *MasterListInfo
	Active               bool
	CalcullatorRegionNum int
	VerifierRegionNum    int
	CalcullatorNet       [MaxCalNetTreeNum]ElectionTree
	VerifierNet          [MaxVerifyNetTreeNum]ElectionTree
	ListMapTreeTbl       [MaxMasterNodeNum]listNodeToTreeNodeInfo
}
func (ml * MasterListInfo) SetTestNode(parentHash uint64) {
	TestNodeNum := 9

	var testNode ListNodeInfo
	//ml := elector.ElectionNet[bufferInd].MasterList
	ml.SetMasterNodeNum(0)
	ml.SetPreHash(parentHash)
	ml.AddMasterNodeInfo(&LocalMasterNodeInfo)

	for nodeLoop := 1; nodeLoop < TestNodeNum; nodeLoop++ {
		testNode.AnnonceRate = uint32(100 + nodeLoop)
		testNode.Ip = "192.168.100." + string(0x30+nodeLoop)
		testNode.NodeId = uint32(nodeLoop)
		testNode.MortgageAccount = uint64(10000 / (nodeLoop + 1))
		testNode.TxHash = uint64(nodeLoop + 10000)
		testNode.Upime = uint64(1000)
		ml.AddMasterNodeInfo(&testNode)
	}
}
/*
func ElectionRun() {
	log.Info(" Election Run")
	go func() {
		for {
			v := <-ChElectionStart
			log.Info("***ELECTION***Election Start", "Block Height :", v.MasterList.masterNum)
			v.Election(v.MasterList)
			log.Info("********************************Computing Power Network******************************")
			for i := 0; i < v.CalcullatorRegionNum; i++ {
				log.Info("|", "============grouping", i, "============")
				for j := 0; j < v.CalcullatorNet[i].Depth; j++ {
					log.Info("|", "***********LEVEL", j)
					node := v.CalcullatorNet[i].GetSpecLevelNode(j)
					for k := 0; k < len(node); k++ {
						log.Info("|", "IP:", node[k].GetSelfIP(v), "Parent IP:", node[k].GetParentIP(v, v.CalcullatorNet[i]))
					}
				}
				log.Info("|", "===============", "============")
			}
			log.Info("********************************Verification Network******************************")
			for i := 0; i < v.VerifierRegionNum; i++ {
				log.Info("|", "============grouping", i, "============")
				for j := 0; j < v.VerifierNet[i].Depth; j++ {
					log.Info("|", "***********LEVEL", j)
					node := v.VerifierNet[i].GetSpecLevelNode(j)
					for k := 0; k < len(node); k++ {
						log.Info("|", "IP:", node[k].GetSelfIP(v), "Parent IP:", node[k].GetParentIP(v, v.VerifierNet[i]))
					}
				}
				log.Info("|", "===============", "============")
			}
		}
	}()
}
*/
func (election *ElectionNetInfo) Init() {

	election.Active = false
	election.CalcullatorRegionNum = 0
	election.VerifierRegionNum = 0

	for i := 0; i < MaxCalNetTreeNum; i++ {
		election.CalcullatorNet[i].Clear()
	}

	for i := 0; i < MaxVerifyNetTreeNum; i++ {
		election.VerifierNet[i].Clear()
	}

	for i := 0; i < MaxMasterNodeNum; i++ {
		election.ListMapTreeTbl[i].regionIndex = 0
		election.ListMapTreeTbl[i].treeNodeIndex = 0
		election.ListMapTreeTbl[i].treeType = CalcullatorNetFlag
	}
}

func (election *ElectionNetInfo) Election(masterList *MasterListInfo) {

	var calcullatorRegionList [MaxCalNetTreeNum][]int
	var verifierRegionList [MaxVerifyNetTreeNum][]int

	election.Init()

	//network not established if nodes are fewer than 3
	if masterList.masterNum < 3 {
		return
	}

	//Grouping based on the number of masternodes: <32; <96; normal
	//Data structure: one dimensional list. From top to bottom; from left to right; what is stored is the index in the masternode list to the nodes
	if masterList.masterNum < MaxTreeNum {
		calcullatorRegionList, verifierRegionList = dividRegionMethodLess32(election, masterList.masterNodeList[0:masterList.masterNum])
	} else if masterList.masterNum < (3 * MaxTreeNum) {
		calcullatorRegionList, verifierRegionList = dividRegionMethodLess96(election, masterList.masterNodeList[0:masterList.masterNum])
	} else {
		calcullatorRegionList, verifierRegionList = dividRegionMethodNormal(election, masterList.masterNodeList[0:masterList.masterNum])
	}

	//Generate binary trees based on the grouping sequence
	for i := 0; i < election.CalcullatorRegionNum; i++ {
		election.CalcullatorNet[i].GenTree(calcullatorRegionList[i][:], i, CalcullatorNetFlag, &election.ListMapTreeTbl)
	}
	for i := 0; i < election.VerifierRegionNum; i++ {
		election.VerifierNet[i].GenTree(verifierRegionList[i][:], i, VerifierNetFlag, &election.ListMapTreeTbl)
	}

	//Update the network topology to reflect dropped nodes
	for i := 0; i < len(masterList.offlineNodeList); i++ {
		//Look in the global list for the index to this node
		if globalIndex, state := masterList.masterNodeSearch(masterList.offlineNodeList[i].TxHash, masterList.offlineNodeList[i].NodeId); state {
			//Refresh the tree
			if election.ListMapTreeTbl[globalIndex].treeType == CalcullatorNetFlag {
				election.CalcullatorNet[election.ListMapTreeTbl[globalIndex].regionIndex].refresh(election.ListMapTreeTbl[globalIndex].treeNodeIndex)
			} else {
				election.VerifierNet[election.ListMapTreeTbl[globalIndex].regionIndex].refresh(election.ListMapTreeTbl[globalIndex].treeNodeIndex)
			}
		}

	}
	election.Active = true

	return
}

func dividRegionMethodLess32(nodeTopologicInfo *ElectionNetInfo, masterNodeList []ListNodeInfo) ([MaxCalNetTreeNum][]int, [MaxVerifyNetTreeNum][]int) {

	nodeLen := len(masterNodeList)

	nodeTopologicInfo.VerifierRegionNum = 0
	nodeTopologicInfo.CalcullatorRegionNum = 0

	sortList := make([]uint32, nodeLen)

	for i := 0; i < nodeLen; i++ {
		sortList[i] = masterNodeList[i].AnnonceRate
	}

	_, sortedIndex := newSort(sortList)
	calcullatorRegionList, verifierRegionList := dividRegion(sortedIndex)

	for i := 0; i < MaxCalNetTreeNum; i++ {
		if len(calcullatorRegionList[i]) > 0 {
			nodeTopologicInfo.CalcullatorRegionNum++
		}
	}
	for i := 0; i < MaxVerifyNetTreeNum; i++ {
		if len(verifierRegionList[i]) > 0 {
			nodeTopologicInfo.VerifierRegionNum++
		}
	}
	return calcullatorRegionList, verifierRegionList
}

func dividRegionMethodLess96(nodeTopologicInfo *ElectionNetInfo, masterNodeList []ListNodeInfo) ([MaxCalNetTreeNum][]int, [MaxVerifyNetTreeNum][]int) {
	nodeLen := len(masterNodeList)

	//All groups will qualify when >=32
	nodeTopologicInfo.VerifierRegionNum = MaxVerifyNetTreeNum
	nodeTopologicInfo.CalcullatorRegionNum = MaxCalNetTreeNum

	sortList := make([]uint32, nodeLen)

	for i := 0; i < nodeLen; i++ {
		sortList[i] = masterNodeList[i].AnnonceRate
	}

	_, sortedIndex := newSort(sortList)
	calcullatorRegionList, verifierRegionList := dividRegion(sortedIndex)

	return calcullatorRegionList, verifierRegionList
}

//Quantization Function: Input: Quantization threshold, Quantization output, original data;
//          Output: Quantization value
func quantization(threshold []uint64, mapval []uint32, val []uint64) []uint32 {

	lenThreshold := len(threshold)
	lenMapVal := len(mapval)
	lenVal := len(val)

	quantiVal := make([]uint32, len(val))
	if (lenThreshold == 0) || (lenMapVal == 0) || (lenVal == 0) {
		return quantiVal
	}

	if lenMapVal != (lenThreshold + 1) {
		return quantiVal
	}

	for i := 0; i < lenVal; i++ {
		quantiVal[i] = mapval[lenMapVal-1]
		for j := 0; j < lenThreshold; j++ {
			if val[i] < threshold[j] {
				quantiVal[i] = mapval[j]
				break
			}
		}
	}

	return quantiVal
}
func swap(a uint32, b uint32) (uint32, uint32) {
	return b, a
}
func swapInt(a int, b int) (int, int) {

	return b, a
}
func dividRegionMethodNormal(nodeTopologicInfo *ElectionNetInfo, masterNodeList []ListNodeInfo) ([MaxCalNetTreeNum][]int, [MaxVerifyNetTreeNum][]int) {
	nodeLen := len(masterNodeList)

	tpsQuantifyThreshold := []uint64{2000, 4000, 8000, 16000}
	tpsQuantifyMap := []uint32{1, 2, 3, 4, 5}
	uptimeQuantifyThreshold := []uint64{65, 129, 257, 513}
	uptimeQuantifyVal := []uint32{16, 8, 4, 2, 1} //real value magnified by 4 times
	depositQuantifyThreshold := []uint64{20000, 40000}
	depositQuantifyVal := []uint32{100, 215, 450} //real value magnified by 100 times

	nodeTopologicInfo.VerifierRegionNum = MaxVerifyNetTreeNum
	nodeTopologicInfo.CalcullatorRegionNum = MaxCalNetTreeNum

	//Quantization TPS Value
	tpsVal := make([]uint64, nodeLen)
	for i := 0; i < nodeLen; i++ {
		tpsVal[i] = uint64(masterNodeList[i].AnnonceRate)
	}
	tpsQuaVal := quantization(tpsQuantifyThreshold, tpsQuantifyMap, tpsVal)

	//Quantization deposit
	depositVal := make([]uint64, nodeLen)
	for i := 0; i < nodeLen; i++ {
		depositVal[i] = uint64(masterNodeList[i].MortgageAccount)
	}
	depositQuaVal := quantization(depositQuantifyThreshold, depositQuantifyVal, depositVal)

	//Quantization update: uptime format not decided yet. Up to 32 bits temporarily used
	uptime := make([]uint32, nodeLen)
	for i := 0; i < nodeLen; i++ {
		uptime[i] = uint32(masterNodeList[i].Upime)
	}
	uptimeOder := upTimeRank(uptime)
	uptimeQuaVal := quantization(uptimeQuantifyThreshold, uptimeQuantifyVal, uptimeOder)

	sortedList := make([]uint32, nodeLen)
	for i := 0; i < nodeLen; i++ {
		sortedList[i] = (tpsQuaVal[i]*4 + depositQuaVal[i]*5) * uptimeQuaVal[i]
	}
	sortedVal, sortedIndex := newSort(sortedList)

	//filter by the same value. Exchange based on TxHash value, the bigger ones come first

	for i := 0; i < nodeLen-1; i++ {
		if (sortedVal[i] == sortedVal[i+1]) && (masterNodeList[sortedIndex[i]].TxHash < masterNodeList[sortedIndex[i+1]].TxHash) {
			sortedVal[i+1], sortedVal[i] = swap(sortedVal[i], sortedVal[i+1])
			sortedIndex[i+1], sortedIndex[i] = swapInt(sortedIndex[i], sortedIndex[i+1])
		}
	}

	calcullatorRegionList, verifierRegionList := dividRegionNorm(sortedIndex, sortedVal, nodeTopologicInfo.MasterList)

	return calcullatorRegionList, verifierRegionList
}

type sortValAndIndex struct {
	val   uint32
	index int
}
type sortValAndIndexSlice []sortValAndIndex

func (c sortValAndIndexSlice) Len() int {
	return len(c)
}
func (c sortValAndIndexSlice) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}
func (c sortValAndIndexSlice) Less(i, j int) bool {
	return c[i].val > c[j].val
}
func newSort(val []uint32) ([]uint32, []int) {
	temp := make([]sortValAndIndex, len(val))
	for i := 0; i < len(val); i++ {
		temp[i].val = val[i]
		temp[i].index = i
	}
	sort.Sort(sortValAndIndexSlice(temp))
	sorted_val := make([]uint32, len(val))
	sorted_index := make([]int, len(val))

	for i := 0; i < len(val); i++ {
		sorted_val[i] = temp[i].val
		sorted_index[i] = temp[i].index
	}
	return sorted_val, sorted_index
}

func upTimeRank(val []uint32) []uint64 {
	temp := make([]sortValAndIndex, len(val))
	for i := 0; i < len(val); i++ {
		temp[i].val = val[i]
		temp[i].index = i
	}
	sort.Sort(sortValAndIndexSlice(temp))
	rank := make([]uint64, len(val))

	for i := 0; i < len(val); i++ {
		rank[temp[i].index] = uint64(i)
	}
	return rank
}

func dividRegion(indexList []int) ([MaxCalNetTreeNum][]int, [MaxVerifyNetTreeNum][]int) {
	nodeLen := len(indexList)

	if nodeLen == 0 {
		log.Info("grouping error. the input node list is null")
	}

	var calcullatorRegionList [MaxCalNetTreeNum][]int
	var verifierRegionList [MaxVerifyNetTreeNum][]int

	wholeLevelNum := int(nodeLen / MaxTreeNum)
	remainderNodeNum := nodeLen - (wholeLevelNum * MaxTreeNum)

	nodeCnt := 0

	//assign whole Level
	for levelLoop := 0; levelLoop < wholeLevelNum; levelLoop++ {
		for calcRegionLoop := 0; calcRegionLoop < MaxCalNetTreeNum; calcRegionLoop++ {
			calcullatorRegionList[calcRegionLoop] = append(calcullatorRegionList[calcRegionLoop], indexList[nodeCnt])
			nodeCnt++
		}
		for verifyRegionLoop := 0; verifyRegionLoop < MaxVerifyNetTreeNum; verifyRegionLoop++ {
			verifierRegionList[verifyRegionLoop] = append(verifierRegionList[verifyRegionLoop], indexList[nodeCnt])
			nodeCnt++
		}
	}
	//assign the remaining nodes based on the proportion of 2:1
	if remainderNodeNum > 0 {
		remainderVerifierNum := int(remainderNodeNum / 3)
		remainderCalcullatorNum := remainderNodeNum - remainderVerifierNum

		for calcRegionLoop := 0; calcRegionLoop < remainderCalcullatorNum; calcRegionLoop++ {
			calcullatorRegionList[calcRegionLoop] = append(calcullatorRegionList[calcRegionLoop], indexList[nodeCnt])
			nodeCnt++
		}
		for verifyRegionLoop := 0; verifyRegionLoop < remainderVerifierNum; verifyRegionLoop++ {
			verifierRegionList[verifyRegionLoop] = append(verifierRegionList[verifyRegionLoop], indexList[nodeCnt])
			nodeCnt++
		}
	}

	return calcullatorRegionList, verifierRegionList
}
func dividRegionNorm(indexList []int, valList []uint32, masterNodeInfo *MasterListInfo) ([MaxCalNetTreeNum][]int, [MaxVerifyNetTreeNum][]int) {
	nodeLen := len(indexList)

	if nodeLen == 0 {
		log.Info("grouping error. the input node list is null")
	}

	var calcullatorRegionList [MaxCalNetTreeNum][]int
	var verifierRegionList [MaxVerifyNetTreeNum][]int
	var calcullatorRegionAccList [MaxCalNetTreeNum][]uint32
	var verifierRegionAccList [MaxVerifyNetTreeNum][]uint32

	wholeLevelNum := int(nodeLen / MaxTreeNum)
	remainderNodeNum := nodeLen - (wholeLevelNum * MaxTreeNum)

	nodeCnt := 0

	//assign whole Level
	for levelLoop := 0; levelLoop < wholeLevelNum; levelLoop++ {
		for calcRegionLoop := 0; calcRegionLoop < MaxCalNetTreeNum; calcRegionLoop++ {
			calcullatorRegionList[calcRegionLoop] = append(calcullatorRegionList[calcRegionLoop], indexList[nodeCnt])
			calcullatorRegionAccList[calcRegionLoop] = append(calcullatorRegionAccList[calcRegionLoop], valList[nodeCnt])
			nodeCnt++
		}
		for verifyRegionLoop := 0; verifyRegionLoop < MaxVerifyNetTreeNum; verifyRegionLoop++ {
			verifierRegionList[verifyRegionLoop] = append(verifierRegionList[verifyRegionLoop], indexList[nodeCnt])
			verifierRegionAccList[verifyRegionLoop] = append(verifierRegionAccList[verifyRegionLoop], valList[nodeCnt])
			nodeCnt++
		}
	}
	//assign the remaining nodes based on the proportion of 2:1
	if remainderNodeNum > 0 {
		remainderVerifierNum := int(remainderNodeNum / 3)
		remainderCalcullatorNum := remainderNodeNum - remainderVerifierNum

		for calcRegionLoop := 0; calcRegionLoop < remainderCalcullatorNum; calcRegionLoop++ {
			calcullatorRegionList[calcRegionLoop] = append(calcullatorRegionList[calcRegionLoop], indexList[nodeCnt])
			calcullatorRegionAccList[calcRegionLoop] = append(calcullatorRegionAccList[calcRegionLoop], valList[nodeCnt])
			nodeCnt++
		}
		for verifyRegionLoop := 0; verifyRegionLoop < remainderVerifierNum; verifyRegionLoop++ {
			verifierRegionList[verifyRegionLoop] = append(verifierRegionList[verifyRegionLoop], indexList[nodeCnt])
			verifierRegionAccList[verifyRegionLoop] = append(verifierRegionAccList[verifyRegionLoop], valList[nodeCnt])
			nodeCnt++
		}
	}

	//resequencing within the group
	var hashAcc uint64
	for calcRegionLoop := 0; calcRegionLoop < MaxCalNetTreeNum; calcRegionLoop++ {
		nodeLen := len(calcullatorRegionList[calcRegionLoop])
		for i := 1; i < nodeLen; i++ {
			calcullatorRegionAccList[calcRegionLoop][i] = calcullatorRegionAccList[calcRegionLoop][i] + calcullatorRegionAccList[calcRegionLoop][i-1]
		}

		hashAcc = 0
		for nodeLoop := 0; nodeLoop < len(calcullatorRegionList[calcRegionLoop]); nodeLoop++ {
			hashAcc = hashAcc + masterNodeInfo.masterNodeList[calcullatorRegionList[calcRegionLoop][nodeLoop]].TxHash
		}
		hashAcc = hashAcc & 0x7FFF

		//Todo:
		rand.Seed(int64(hashAcc + (masterNodeInfo.preHash << 15)))
		randdata := uint32(rand.Float32() * float32(calcullatorRegionAccList[calcRegionLoop][nodeLen-1]))
		_, topIndex := searchMinErr(calcullatorRegionAccList[calcRegionLoop], randdata)
		//fmt.Printf("A:%d, %d\n", randdata, topIndex)
		//fmt.Printf("B:%d, %d\n", calcullatorRegionList[calcRegionLoop][0], calcullatorRegionList[calcRegionLoop][topIndex])
		c := calcullatorRegionList[calcRegionLoop][0]
		calcullatorRegionList[calcRegionLoop][0] = calcullatorRegionList[calcRegionLoop][topIndex]
		calcullatorRegionList[calcRegionLoop][topIndex] = c
		//calcullatorRegionList[calcRegionLoop][topIndex], calcullatorRegionList[calcRegionLoop][0] = swapInt(calcullatorRegionList[calcRegionLoop][0], calcullatorRegionList[calcRegionLoop][topIndex])
		//fmt.Printf("C:%d, %d\n", calcullatorRegionList[calcRegionLoop][0], calcullatorRegionList[calcRegionLoop][topIndex])
	}

	for verifyRegionLoop := 0; verifyRegionLoop < MaxVerifyNetTreeNum; verifyRegionLoop++ {
		nodeLen := len(verifierRegionAccList[verifyRegionLoop])
		for i := 1; i < nodeLen; i++ {
			verifierRegionAccList[verifyRegionLoop][i] = verifierRegionAccList[verifyRegionLoop][i] + verifierRegionAccList[verifyRegionLoop][i-1]
		}

		hashAcc = 0
		for nodeLoop := 0; nodeLoop < len(verifierRegionList[verifyRegionLoop]); nodeLoop++ {
			hashAcc = hashAcc + masterNodeInfo.masterNodeList[calcullatorRegionList[verifyRegionLoop][nodeLoop]].TxHash
		}
		hashAcc = hashAcc & 0x7FFF

		rand.Seed(int64(hashAcc + (masterNodeInfo.preHash << 15)))
		randdata := uint32(rand.Float32() * float32(verifierRegionAccList[verifyRegionLoop][nodeLen-1]))
		_, topIndex := searchMinErr(verifierRegionAccList[verifyRegionLoop], randdata)
		c := verifierRegionList[verifyRegionLoop][0]
		verifierRegionList[verifyRegionLoop][0] = calcullatorRegionList[verifyRegionLoop][topIndex]
		verifierRegionList[verifyRegionLoop][topIndex] = c
		//verifierRegionList[verifyRegionLoop][topIndex], verifierRegionList[verifyRegionLoop][0] = swapInt(verifierRegionList[verifyRegionLoop][0], verifierRegionList[verifyRegionLoop][topIndex])
	}

	return calcullatorRegionList, verifierRegionList
}

func searchMinErr(array []uint32, benchmark uint32) (uint32, int) {
	arrayLen := len(array)

	diff := func(a uint32, b uint32) int {
		if a > b {
			return int(a - b)
		} else {
			return int(b - a)
		}
	}
	if arrayLen == 0 {
		log.Info("searchMinErr, the length of the input slice is 0")
		return 0, 0
	}

	index := 0
	errVal := diff(array[0], benchmark)
	for i := 1; i < arrayLen; i++ {
		t := diff(array[i], benchmark)
		if t < errVal {
			errVal = t
			index = i
		}
	}

	return array[index], index
}
