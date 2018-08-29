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

package ca

import (
	"errors"
	"math/big"
	"sync"

	"github.com/syndtr/goleveldb/leveldb"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/depoistInfo"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/mc"
	"github.com/ethereum/go-ethereum/p2p/discover"
)

// Identity stand for node's identity.
type Identity struct {
	// self nodeId
	self discover.NodeID
	addr common.Address

	// if in elected duration
	duration      bool
	currentHeight *big.Int

	// levelDB
	ldb *leveldb.DB

	// self previous, current and next role type
	prvRole     common.RoleType
	currentRole common.RoleType
	nextRole    common.RoleType

	// chan to listen block coming and quit message
	blockChan chan *types.Block
	quit      chan struct{}

	// lock and once to sync
	lock *sync.RWMutex
	once *sync.Once

	// sub to unsubscribe block channel
	sub event.Subscription

	// logger
	log log.Logger

	// save id list
	availableId uint32
	idList      map[discover.NodeID]uint32

	// elect result: [role type, [nodeId]]
	elect map[common.Address]common.RoleType
	// current topology: [role type, [nodeId]]
	topology map[uint16]common.Address

	// next elect result: [role type, [nodeId]]
	nextElect map[common.Address]common.RoleType

	// addrByGroup
	addrByGroup map[common.RoleType][]common.Address
}

var Ide = newIde()

func newIde() *Identity {
	return &Identity{
		quit:        make(chan struct{}),
		currentRole: common.RoleNil,
		prvRole:     common.RoleNil,
		nextRole:    common.RoleNil,
		duration:    false,
		lock:        new(sync.RWMutex),
		once:        new(sync.Once),
		idList:      make(map[discover.NodeID]uint32),
		elect:       make(map[common.Address]common.RoleType),
		topology:    make(map[uint16]common.Address),
		nextElect:   make(map[common.Address]common.RoleType),
	}
}
	
// init to do something before run.
func (ide *Identity) init(id discover.NodeID) {
	ide.once.Do(func() {
		// check bootNode and set identity
		ide.self = id
		//ide.ldb, _ = leveldb.OpenFile("./db/ca", nil)
		ide.log = log.New()
		ide.log.Info("identity init over")
	})
}

// Run this Identity.
func (ide *Identity) Start(id discover.NodeID) {
	ide.init(id)

	defer func() {
		ide.sub.Unsubscribe()

		close(ide.quit)
		close(ide.blockChan)
	}()

	ide.blockChan = make(chan *types.Block)
	ide.sub, _ = mc.SubscribeEvent(mc.NewBlockMessage, ide.blockChan)

	for {
		select {
		case block := <-ide.blockChan:

			header := block.Header()
			ide.currentHeight = header.Number
			switch {
			// validator elected block
			case (header.Number.Uint64()+VerifyNetChangeUpTime)%ReelectionInterval == 0:
				{
					ide.duration = true
					// maintain topology and check self next role
					for _, e := range header.Elect {
						ide.nextElect[e.Account] = e.Type
						if e.Account == ide.addr {
							ide.nextRole = e.Type
						}
					}
				}
			// miner elected block
			case (header.Number.Uint64()+MinerNetChangeUpTime)%ReelectionInterval == 0:
				{
					ide.duration = true

					for _, e := range header.Elect {
						ide.nextElect[e.Account] = e.Type
						if e.Account == ide.addr {
							ide.nextRole = e.Type
						}
					}
				}
			// formal elected block
			case header.Number.Uint64()%ReelectionInterval == 0:
				{
					ide.duration = false

					ide.elect = ide.nextElect
					ide.nextElect = make(map[common.Address]common.RoleType)

					ide.prvRole = ide.nextRole
					for _, e := range header.Elect {
						if e.Account == ide.addr {
							ide.currentRole = e.Type
							ide.nextRole = common.RoleNil
						}
					}
				}
			case header.Number.Uint64() == 0:
				{
					for _, e := range header.Elect {
						ide.nextElect[e.Account] = e.Type
						if e.Account == ide.addr {
							ide.currentRole = e.Type
						}
					}
				}
			default:
			}

			// do topology
			ide.InitCurrentTopology(header.NetTopology)

			// change default role
			if ide.currentRole == common.RoleNil {
				ide.currentRole = common.RoleDefault
			}

			// init now topology: self peer
			ide.initNowTopologyResult()

			// get nodes in buckets and send to buckets
			nodesInBuckets := ide.getNodesInBuckets(block.Header().Number)
			mc.PublicEvent(mc.BlockToBuckets, mc.BlockToBucket{Ms: nodesInBuckets, Height: block.Header().Number, Role: ide.currentRole})
			// send identity to linker
			mc.PublicEvent(mc.BlockToLinkers, mc.BlockToLinker{Height: header.Number, Role: ide.currentRole, RoleNext: ide.nextRole})
		case <-ide.quit:
			return
		}
	}
}

// Stop this Identity.
func (ide *Identity) Stop() {
	ide.log.Info("identity stop")

	ide.lock.Lock()
	ide.quit <- struct{}{}
	ide.lock.Unlock()
}

// InitCurrentTopology init current topology.
func (ide *Identity) InitCurrentTopology(tp common.NetTopology) {
	switch tp.Type {
	case 1:
		for _, v := range tp.NetTopologyData {
			ide.topology[v.Position] = v.Account
		}
	case 0:
		ide.topology = make(map[uint16]common.Address)
		for _, v := range tp.NetTopologyData {
			ide.topology[v.Position] = v.Account
		}
	}
}

// initNowTopologyResult
func (ide *Identity) initNowTopologyResult() {
	ide.addrByGroup = make(map[common.RoleType][]common.Address)
	for po, addr := range ide.topology {
		role := common.GetRoleTypeFromPosition(po)
		ide.addrByGroup[role] = append(ide.addrByGroup[role], addr)
	}
}

// GetRolesByGroup
func (ide *Identity) GetRolesByGroup(roleType common.RoleType) (result []discover.NodeID) {
	for k, v := range ide.addrByGroup {
		if (k & roleType) != 0 {
			for _, addr := range v {
				id, _ := discover.HexID(addr.Hex())
				result = append(result, id)
			}
		}
	}
	return
}

// Get self identity.
func (ide *Identity) GetRole() (role common.RoleType) {
	ide.lock.Lock()
	defer ide.lock.Unlock()

	return ide.currentRole
}

func (ide *Identity) GetHeight() *big.Int {
	ide.lock.Lock()
	defer ide.lock.Unlock()

	return ide.currentHeight
}

// InDuration
func (ide *Identity) InDuration() bool {
	ide.lock.Lock()
	defer ide.lock.Unlock()

	return ide.duration
}

// GetElectedByHeightAndRole get elected node, miner or validator by block height and type.
func (ide *Identity) GetElectedByHeightAndRole(height *big.Int, roleType common.RoleType) ([]vm.DepositDetail, error) {
	return depoistInfo.GetDepositList(height, roleType)
}

// GetElectedByHeight get all elected node by height.
func (ide *Identity) GetElectedByHeight(height *big.Int) ([]vm.DepositDetail, error) {
	return depoistInfo.GetAllDeposit(height)
}

// GetNodeNumber
func (ide *Identity) GetNodeNumber() (uint32, error) {
	for k, v := range ide.idList {
		if k == ide.self {
			return v, nil
		}
	}
	return 0, errors.New("No current node number. ")
}

// getNodesInBuckets get miner nodes that should be in buckets.
func (ide *Identity) getNodesInBuckets(height *big.Int) (result []discover.NodeID) {
	electedMiners, _ := ide.GetElectedByHeightAndRole(height, common.RoleMiner)

	msMap := make(map[common.Address]discover.NodeID)
	for _, m := range electedMiners {
		msMap[m.Address] = m.NodeID
	}
	for _, v := range ide.topology {
		for key := range msMap {
			if key == v {
				delete(msMap, key)
				break
			}
		}
	}
	for key, val := range msMap {
		if ide.addr == key {
			ide.currentRole = common.RoleBucket
		}
		result = append(result, val)
	}
	return
}

// GetTopologyInLinker
func (ide *Identity) GetTopologyInLinker() (result map[common.RoleType][]discover.NodeID) {
	for k, v := range ide.addrByGroup {
		for _, addr := range v {
			id, _ := discover.HexID(addr.Hex())
			result[k] = append(result[k], id)
		}
	}
	return
}
