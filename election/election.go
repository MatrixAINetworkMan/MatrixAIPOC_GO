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
