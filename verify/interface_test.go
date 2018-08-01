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
package verifier

import (
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/election"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"math/big"
	"math/rand"
	"testing"
	"time"
)

type testStruct struct {
	IP   string
	ID   string
	Type uint32
}

func prepareTxList() []*types.Transaction {

	testList := make([]testStruct, 0)
	testList = append(testList, testStruct{IP: "192.168.3.95", ID: "27044ec5f811cd6d1a9410ab539bf68efdc0ea0343bf437500fe176494cd18438f18be6a3f11efda7a4955c3b03ffa7810e3b121efd9b843bf39aabb72c1bf6f", Type: types.ElectMiner})

	toAddress := common.HexToAddress("2c8b487d9c75d009d9e9cb8291952d715c36324c")
	toAddressIllegal := common.HexToAddress("7f56487d9c75d009d9e9cb8291952d715c36328d")

	log.Info("hyk test: right to address", "address", toAddress)
	log.Info("hyk test: illegal to address", "address", toAddressIllegal)

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	txList := make([]*types.Transaction, 0)
	for i, temp := range testList {
		payLoad := types.ElectionTxPayLoadInfo{TPS: 2000, IP: temp.IP, ID: temp.ID, Wealth: 20000, OnlineTime: 1000, TxHash: uint64(i), Value: 0, ElectType: temp.Type}

		data := make([]byte, 0)
		switch temp.Type {
		case types.ElectExit:
			data = append(data, 170)
		case types.ElectMiner:
			data = append(data, 255)
		case types.ElectCommittee:
			data = append(data, 238)
		case types.ElectBoth:
			data = append(data, 221)
		}

		jsonData, err := json.Marshal(payLoad)
		if err != nil {
			log.Warn("hyk test: json Marshal err", err)
			continue
		}

		data = append(data, jsonData...)

		fmt.Printf("data: [%x]\n", string(data))

		var tx *types.Transaction
		temp := r.Intn(10)
		if temp < 10 {
			tx = types.NewTransaction(321, toAddress, big.NewInt(3000), 4000, big.NewInt(200), data)
		} else {
			tx = types.NewTransaction(321, toAddressIllegal, big.NewInt(3000), 4000, big.NewInt(200), data)
		}

		if nil == tx {
			continue
		}

		fmt.Println("hyk test: add tx, payload", payLoad)
		txList = append(txList, tx)
	}

	return txList
}

func prepareBlkMList() (nodeList election.NodeList) {
	testList := make([]testStruct, 0)
	testList = append(testList, testStruct{ID: "test_11", Type: types.ElectCommittee})
	testList = append(testList, testStruct{ID: "test_12", Type: types.ElectCommittee})
	testList = append(testList, testStruct{ID: "test_13", Type: types.ElectMiner})
	testList = append(testList, testStruct{ID: "test_14", Type: types.ElectMiner})
	testList = append(testList, testStruct{ID: "test_15", Type: types.ElectMiner})
	testList = append(testList, testStruct{ID: "test_16", Type: types.ElectBoth})
	testList = append(testList, testStruct{ID: "test_17", Type: types.ElectBoth})
	testList = append(testList, testStruct{ID: "test_18", Type: types.ElectExit})
	testList = append(testList, testStruct{ID: "test_19", Type: types.ElectExit})

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for _, temp := range testList {
		info := election.NodeInfo{TPS: r.Uint32() % 100, IP: "192.168.3.3", ID: temp.ID, Wealth: r.Uint64() % 1000, OnlineTime: r.Uint64() % 3000, TxHash: r.Uint64() % 3000, Value: r.Uint64() % 3000}
		switch temp.Type {
		case types.ElectExit:
			nodeList.OfflineList = append(nodeList.OfflineList, info)
		case types.ElectMiner:
			nodeList.MinerList = append(nodeList.MinerList, info)
		case types.ElectCommittee:
			nodeList.CommitteeList = append(nodeList.CommitteeList, info)
		case types.ElectBoth:
			nodeList.Both = append(nodeList.Both, info)
		}
	}

	fmt.Println("hyk test: Last MList exit", nodeList.OfflineList)
	fmt.Println("hyk test: Last MList miner", nodeList.MinerList)
	fmt.Println("hyk test: Last MList committee", nodeList.CommitteeList)
	fmt.Println("hyk test: Last MList both", nodeList.Both)

	return nodeList
}

func TestInterface(t *testing.T) {
	fmt.Println("hyk test: begin!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")

	//do test
	nodeList, _ := GenerateMainNodeListfoForTest(479, prepareBlkMList(), prepareTxList())

	fmt.Println("hyk test: result", "exit node list", nodeList.OfflineList)
	fmt.Println("hyk test: result", "miner node list", nodeList.MinerList)
	fmt.Println("hyk test: result", "committee node list", nodeList.CommitteeList)
	fmt.Println("hyk test: result", "both node list", nodeList.Both)
}

func GenerateMainNodeListfoForTest(currentNumber uint64, lastBlkMList election.NodeList, txList []*types.Transaction) (election.NodeList, error) {
	var returnList election.NodeList

	if (currentNumber+1)%params.BroadcastInterval != 0 {
		fmt.Println("block number = ", currentNumber, ", it is not the time to generate main node list")
		return returnList, fmt.Errorf("block number = %d, it is not the time to generate main node list", currentNumber)
	}

	var minerList, committeeList, bothList []election.NodeInfo
	lastBroadcastBlkNumber := currentNumber + 1 - params.BroadcastInterval
	if lastBroadcastBlkNumber == 0 {
		minerList = nil
		committeeList = nil
		bothList = nil
	} else {
		minerList = lastBlkMList.MinerList
		committeeList = lastBlkMList.CommitteeList
		bothList = lastBlkMList.Both
	}

	newNodeMap, err := GetElectionAndExitNodeInfoForTest(txList)
	if err != nil {
		return returnList, err
	}

	// 将上个广播区块中的主节点列表和本周期内出现的新参选退选合并，其中上个广播区块中退选列表不需要合并
	mergeNodeInfo(newNodeMap, minerList, types.ElectMiner)
	mergeNodeInfo(newNodeMap, committeeList, types.ElectCommittee)
	mergeNodeInfo(newNodeMap, bothList, types.ElectBoth)

	// 由合并后的map生产本周期的主节点列表
	for _, value := range newNodeMap {
		info := election.NodeInfo{
			TPS:        value.TPS,
			IP:         value.IP,
			ID:         value.ID,
			Wealth:     value.Wealth,
			OnlineTime: value.OnlineTime,
			TxHash:     value.TxHash,
			Value:      value.Value,
			Account:    value.Account,
		}

		switch value.ElectType {
		case types.ElectExit:
			returnList.OfflineList = append(returnList.OfflineList, info)
		case types.ElectMiner:
			returnList.MinerList = append(returnList.MinerList, info)
		case types.ElectCommittee:
			returnList.CommitteeList = append(returnList.CommitteeList, info)
		case types.ElectBoth:
			returnList.Both = append(returnList.Both, info)
		}
	}

	return returnList, nil
}

func GetElectionAndExitNodeInfoForTest(txList []*types.Transaction) (map[string]*types.ElectionTxPayLoadInfo, error) {
	if len(txList) == 0 {
		return nil, fmt.Errorf("input block number index err")
	}

	infoMap := make(map[string]*types.ElectionTxPayLoadInfo)

	//todo 可能需要添加，一个节点出现多次交易时，累加交易金额。考虑中间出现退出时，清空的情况
	for _, tx := range txList {
		if nil == tx {
			continue
		}

		nodeInfo := tx.ParseElectionTxPayLoad()
		if nodeInfo == nil {
			continue
		}

		infoMap[nodeInfo.ID] = nodeInfo
	}

	return infoMap, nil
}
