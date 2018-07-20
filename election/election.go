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

import "github.com/ethereum/go-ethereum/common"

type NodeList struct {
	MinerList     []NodeInfo	`json:"miner_list"`
	CommitteeList []NodeInfo	`json:"committee_list"`
	Both          []NodeInfo	`json:"both"`
	OfflineList   []NodeInfo	`json:"offline_list"`
}
type NodeInfo struct {
	TPS        uint32			`json:"tps"`
	IP         string			`json:"ip"`
	ID         string			`json:"id"`
	Wealth     uint64			`json:"wealth"`
	OnlineTime uint64			`json:"online_time"`
	TxHash     uint64			`json:"tx_hash"`
	Value      uint64			`json:"value"`
	Account	   common.Address	`json:"account"`
}

type ElectionEngine interface {
	GenNetwork(n NodeList, networkGenerated chan<- bool)
	GetChild(id string) []NodeInfo
	GetParent(id string) NodeInfo
	GetLeafNode(id string) NodeInfo
	GetIDType(id string) int
	GetSuperMiner() []NodeInfo
	GetSuperCommittee() []NodeInfo
}
