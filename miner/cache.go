package miner

import (
	"sync"
	"github.com/ethereum/go-ethereum/election"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/core/types"
)

type broadcastCache struct {
	NodeList	election.NodeList
	BlockNum 	uint64
	mu 			sync.Mutex
}

func newBroadCastCache() *broadcastCache {
	cache := &broadcastCache{
		NodeList:	election.NodeList{},
		BlockNum:	0,
	}

	// using boot info init the 0 block node list
	cache.NodeList.CommitteeList = make([]election.NodeInfo, 0)
	if committeeNode, err := discover.ParseNode(params.MainnetBootnodes[0]); err == nil {
		cache.NodeList.CommitteeList = append(cache.NodeList.CommitteeList, election.NodeInfo{ID:committeeNode.ID.String(), IP: committeeNode.IP.String()})
	}

	cache.NodeList.MinerList = make([]election.NodeInfo, 0)
	for i := 1; i < len(params.MainnetBootnodes); i++ {
		if bootNode, err := discover.ParseNode(params.MainnetBootnodes[i]); err == nil {
			cache.NodeList.MinerList = append(cache.NodeList.MinerList, election.NodeInfo{ID:bootNode.ID.String(), IP: bootNode.IP.String()})
		}
	}

	return cache
}

func (cache *broadcastCache) update(info *BroadcastInfo)  {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	log.Info("worker log: receive broadcast info", "curNumber", cache.BlockNum, "infoNumber", info.BlockNum,
		"committee", len(info.NodeList.CommitteeList), "miner", len(info.NodeList.MinerList), "both", len(info.NodeList.Both), "offline", len(info.NodeList.OfflineList))

	if info.BlockNum <= cache.BlockNum {
		return
	}

	if (info.BlockNum+2)%params.BroadcastInterval != 0 {
		log.Warn("worker log: the number of broadcast info is not OK!!", "blockNumber", info.BlockNum, "BCInterval", params.BroadcastInterval)
		return
	}

	cache.NodeList.MinerList = make([]election.NodeInfo, 0, len(info.NodeList.MinerList))
	cache.NodeList.CommitteeList = make([]election.NodeInfo, 0, len(info.NodeList.CommitteeList))
	cache.NodeList.Both = make([]election.NodeInfo, 0, len(info.NodeList.Both))
	cache.NodeList.OfflineList = make([]election.NodeInfo, 0, len(info.NodeList.OfflineList))

	cache.NodeList.MinerList = append(cache.NodeList.MinerList, info.NodeList.MinerList...)
	cache.NodeList.CommitteeList = append(cache.NodeList.CommitteeList, info.NodeList.CommitteeList...)
	cache.NodeList.Both = append(cache.NodeList.Both, info.NodeList.Both...)
	cache.NodeList.OfflineList = append(cache.NodeList.OfflineList, info.NodeList.OfflineList...)

	cache.BlockNum = info.BlockNum + 2

	log.Info("worker log: broadcast cache info updated", "minerList", cache.NodeList.MinerList, "committeeList",
		cache.NodeList.CommitteeList, "BothList", cache.NodeList.Both, "offlineList", cache.NodeList.OfflineList)

	return
}

func  (cache *broadcastCache) writeCacheToHeader(h *types.Header) {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	headNumber := h.Number.Uint64()
	if headNumber == cache.BlockNum {
		// write broadcast block with main node list info
		cache.copyNodeListToHeader(h)
	} else {
		log.Warn("broadcastCache log: broadcastCache is incorrect", "curNumber", headNumber, "cacheNumber", cache.BlockNum)
		//todo 异常情况处理,暂时使用旧的的广播信息组建广播区块
		cache.copyNodeListToHeader(h)
	}

	return
}

func (cache *broadcastCache) copyNodeListToHeader(h *types.Header) {
	h.MinerList = make([]election.NodeInfo, 0, len(cache.NodeList.MinerList))
	h.MinerList = append(h.MinerList, cache.NodeList.MinerList...)

	h.CommitteeList = make([]election.NodeInfo, 0, len(cache.NodeList.CommitteeList))
	h.CommitteeList = append(h.CommitteeList, cache.NodeList.CommitteeList...)

	h.Both = make([]election.NodeInfo, 0, len(cache.NodeList.Both))
	h.Both = append(h.Both, cache.NodeList.Both...)

	h.OfflineList = make([]election.NodeInfo, 0, len(cache.NodeList.OfflineList))
	h.OfflineList = append(h.OfflineList, cache.NodeList.OfflineList...)
}