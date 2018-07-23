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
	"fmt"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/election"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/params"
)

func newPayLoadInfo(info *election.NodeInfo, electType uint32) *types.ElectionTxPayLoadInfo {
	return &types.ElectionTxPayLoadInfo{
		TPS:        info.TPS,
		IP:         info.IP,
		ID:         info.ID,
		Wealth:     info.Wealth,
		OnlineTime: info.OnlineTime,
		TxHash:     info.TxHash,
		Value:      info.Value,
		ElectType:  electType,
		Account:    info.Account,
	}
}

func mergeNodeInfo(nodeMap map[string]*types.ElectionTxPayLoadInfo, nodeList []election.NodeInfo, electType uint32) {
	if nil == nodeList {
		return
	}

	for _, nodeInfo := range nodeList {
		if _, ok := nodeMap[nodeInfo.ID]; ok{
			// already exist!
			continue
		} else {
			nodeMap[nodeInfo.ID] = newPayLoadInfo(&nodeInfo, electType)
		}
	}
}

func (v *Verifier)GenerateMainNodeList(currentNumber uint64) (election.NodeList, error) {
	var returnList election.NodeList

	if (currentNumber + 2) % params.BroadcastInterval != 0 {
		return returnList, fmt.Errorf("block number = %d, it is not the time to generate main node list", currentNumber)
	}

	var minerList, committeeList, bothList []election.NodeInfo
	lastBroadcastBlkNumber := currentNumber + 2 - params.BroadcastInterval
	if lastBroadcastBlkNumber == 0 {
		minerList = nil
		committeeList = nil
		bothList = nil

		// using boot info init the 0 block node list
		committeeList = make([]election.NodeInfo, 0)
		committeeURL := params.MainnetBootnodes[len(params.MainnetBootnodes) - 1]
		if committeeNode, err := discover.ParseNode(committeeURL); err == nil {
			committeeList = append(committeeList, election.NodeInfo{ID:committeeNode.ID.String(), IP: committeeNode.IP.String()})
		}

		minerList = make([]election.NodeInfo, 0)
		for i := 0; i < len(params.MainnetBootnodes) - 1; i++ {
			if bootNode, err := discover.ParseNode(params.MainnetBootnodes[i]); err == nil {
				minerList = append(minerList, election.NodeInfo{ID:bootNode.ID.String(), IP: bootNode.IP.String()})
			}
		}

	} else {
		lastBroadcastBlk := v.chain.GetBlockByNumber(lastBroadcastBlkNumber)
		if nil == lastBroadcastBlk {
			return returnList, fmt.Errorf("get last broadcast block(%d) err", lastBroadcastBlkNumber)
		}

		minerList = lastBroadcastBlk.Header().MinerList
		committeeList = lastBroadcastBlk.Header().CommitteeList
		bothList = lastBroadcastBlk.Header().Both
	}

	var startPos uint64
	if currentNumber > params.BroadcastInterval - 1 {
		startPos = currentNumber - params.BroadcastInterval + 1
	} else {
		startPos = 0
	}

	newNodeMap, err := v.GetElectionAndExitNodeInfo(startPos, currentNumber)
	if err != nil {
		return  returnList, err
	}

	// merge node info list with last broadcast info
	mergeNodeInfo(newNodeMap, minerList, types.ElectMiner)
	mergeNodeInfo(newNodeMap, committeeList, types.ElectCommittee)
	mergeNodeInfo(newNodeMap, bothList, types.ElectBoth)

	// Generate main node list
	for _, value := range newNodeMap {
		info := election.NodeInfo {
			TPS:value.TPS,
			IP:value.IP,
			ID:value.ID,
			Wealth:value.Wealth,
			OnlineTime:value.OnlineTime,
			TxHash:value.TxHash,
			Value:value.Value,
			Account:value.Account,
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

func (v *Verifier)GetElectionAndExitNodeInfo(blkNumStart uint64, blkNumEnd uint64) (map[string]*types.ElectionTxPayLoadInfo, error) {
	if blkNumStart > blkNumEnd {
		return nil, fmt.Errorf("input block number index err")
	}

	infoMap := make(map[string]*types.ElectionTxPayLoadInfo)

	pos := blkNumStart
	for ; pos <= blkNumEnd; pos++ {
		block := v.chain.GetBlockByNumber(pos)
		if nil == block {
			continue
		}

		for _, tx := range block.Transactions() {
			if nil == tx {
				continue
			}

			nodeInfo := tx.ParseElectionTxPayLoad()
			if nodeInfo == nil {
				continue
			}

			//get from account address
			config := v.chain.Config()
			height := v.chain.CurrentBlock().Header().Number
			fromAddress, err := types.Sender(types.MakeSigner(config, height), tx)
			if err != nil {
				log.Warn("Get from account address err!")
				continue
			}
			nodeInfo.Account = fromAddress

			infoMap[nodeInfo.ID] = nodeInfo
		}
	}

	return infoMap, nil
}
