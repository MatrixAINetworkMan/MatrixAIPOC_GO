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
	"encoding/json"
	"errors"
	"math/big"
	"sync"

	"github.com/syndtr/goleveldb/leveldb"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
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
var Account2Name map[common.Address]int
var Validatoraccountlist = []string{"0x0ead6cdb8d214389909a535d4ccc21a393dddba9", "0x6a3217d128a76e4777403e092bde8362d4117773", "0x0a3f28de9682df49f9f393931062c5204c2bc404", "0x8c3d1a9504a36d49003f1652fadb9f06c32a4408", "0x05e3c16931c6e578f948231dca609d754c18fc09", "0x55fbba0496ef137be57d4c179a3a74c4d0c643be", "0x915b5295dde0cebb11c6cb25828b546a9b2c9369", "0x92e0fea9aba517398c2f0dd628f8cfc7e32ba984", "0x7eb0bcd103810a6bf463d6d230ebcacc85157d19", "0xcded44bd41476a69e8e68ba8286952c414d28af7", "0x9cde10b889fca53c0a560b90b3cb21c2fc965d2b"}
var mineraccountlist = []string{"0x7823a1bea7aca2f902b87effdd4da9a7ef1fc5fb"}
var node2account = map[string]string{"8ce7defe2dde8297f7b55dd9ba8c5e13e0274371b716250ea0dd725974fa076ca379fc7226789a91678f4e38f8f60f8e6405ec9539cab77d4822614e80f743cf": "0x55fbba0496ef137be57d4c179a3a74c4d0c643be",
	"b624a3fb585a48b4c96e4e6327752b1ba82a90a948f258be380ba17ead7c01f6d4ad43d665bb11c50475c058d3aad1ba9a35c0e0c4aa118503bf3ce79609bef6": "0x0ead6cdb8d214389909a535d4ccc21a393dddba9",
	"dbf8dcc4c82eb2ea2e1350b0ea94c7e29f5be609736b91f0faf334851d18f8de1a518def870c774649db443fbce5f72246e1c6bc4a901ef33429fdc3244a93b3": "0x6a3217d128a76e4777403e092bde8362d4117773",
	"a9f94b62067e993f3f02ada1a448c70ae90bdbe4c6b281f8ff16c6f4832e0e9aba1827531c260b380c776938b9975ac7170a7e822f567660333622ea3e529313": "0x0a3f28de9682df49f9f393931062c5204c2bc404",
	"80606b6c1eecb8ce91ca8a49a5a183aa42f335eb0d8628824e715571c1f9d1d757911b80ebc3afab06647da228f36ecf1c39cb561ef7684467c882212ce55cdb": "0x8c3d1a9504a36d49003f1652fadb9f06c32a4408",
	"43b553fae2184b25e76b69a2386bfc9a014486db7da3df75bba9fa2e3eed8aaf063a5f1aab68488a8645fd6a230a27bfe4e8d3393232fe107ba0f68a9bf541ad": "0x05e3c16931c6e578f948231dca609d754c18fc09",
	"9f237f9842f70b0417d2c25ce987248c991310b2bd4034e300a6eec46b517bd8c4f7f31f157128d0732786181a481bcf725c41a655bdcce282a4bc95638d9aae": "0x915b5295dde0cebb11c6cb25828b546a9b2c9369",
	"68315573b123b44367f9fefcce38c4d5c4d5d2caf04158a9068de2060380b81f26b220543de7402745160141f932012a792722fd4dd2a7a2751771097eeef5f2": "0x92e0fea9aba517398c2f0dd628f8cfc7e32ba984",
	"bc5e761c9d0ba42f22433be14973b399662456763f033a4cdbb8ec37b80266526e6c56f92d0591825c7d644e487fcee828d537c58ce583a72578309ec6ebbd39": "0x7eb0bcd103810a6bf463d6d230ebcacc85157d19",
	"25ea3bca7679192612aed14d5e83a4f2a30824ff2af705d2d7c6795470f9cbbc258d9b102a726c3982cda6c4732ba3715551b6fbf9c0ae4ddca4a6c80bc4bbe9": "0xcded44bd41476a69e8e68ba8286952c414d28af7",
	"14f62dfd8826734fe75120849e11614b0763bc584fba4135c2f32b19501525d55d217742893801ecc871023fc42ed7e80196357fb5b1f762d181e827e626637d": "0x9cde10b889fca53c0a560b90b3cb21c2fc965d2b",
	"df57387d6505d0f71d7000da9642cf16d44feb7fcaa5f3a8a7d9fa58b6cbb6d33d145746d4fb544c049d3ff9b534bf9245a5b8052231c51695fd298032bd4a79": "0x7823a1bea7aca2f902b87effdd4da9a7ef1fc5fb",
}

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
		ide.ldb, _ = leveldb.OpenFile("./db/ca", nil)
		ide.log = log.New()
		ide.log.Info("identity init over")

		key, err := id.Pubkey()
		if err != nil {
			ide.log.Error("address error:", "ca", err)
			return
		}
		ide.addr = crypto.PubkeyToAddress(*key)
	})
}

// Run this Identity.
func (ide *Identity) Start(id discover.NodeID) {
	ide.init(id)

	defer func() {
		ide.ldb.Close()
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

			// set to levelDB
			setToLevelDB(header.Elect)

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

func setToLevelDB(elect []common.Elect) error {
	var (
		tg = new(mc.TopologyGraph)
		tp = make([]mc.TopologyNodeInfo, 0)
	)

	tg.Number = Ide.GetHeight()

	for key, val := range Ide.topology {
		st := mc.TopologyNodeInfo{}
		for _, e := range elect {
			if e.Account == val {
				st.Stock = e.Stock
				break
			}
		}
		st.Type = common.GetRoleTypeFromPosition(key)
		st.Account = val
		st.Position = key
		tp = append(tp, st)
	}
	tg.NodeList = tp

	bytes, err := json.Marshal(tg)
	if err != nil {
		return err
	}

	Ide.ldb.Put(tg.Number.Bytes(), bytes, nil)

	return nil
}

// GetAddress
func GetAddress() common.Address {
	Ide.lock.Lock()
	defer Ide.lock.Unlock()

	return Ide.addr
}

// GetTopologyByNumber
func GetTopologyByNumber(reqTypes common.RoleType, number uint64) (*mc.TopologyGraph, error) {
	val, err := Ide.ldb.Get(big.NewInt(int64(number)).Bytes(), nil)
	if err != nil {
		return nil, err
	}

	es := new(mc.TopologyGraph)
	err = json.Unmarshal(val, &es)
	if err != nil {
		return nil, err
	}

	newEs := new(mc.TopologyGraph)
	newEs.Number = es.Number
	for _, tg := range es.NodeList {
		if (tg.Type & reqTypes) != 0 {
			newEs.NodeList = append(newEs.NodeList, tg)
		}
	}

	return newEs, nil
}

// GetAccountTopologyInfo
func GetAccountTopologyInfo(account common.Address, number uint64) (*mc.TopologyNodeInfo, error) {
	val, err := Ide.ldb.Get(big.NewInt(int64(number)).Bytes(), nil)
	if err != nil {
		return nil, err
	}

	es := new(mc.TopologyGraph)
	err = json.Unmarshal(val, &es)
	if err != nil {
		return nil, err
	}

	for _, tg := range es.NodeList {
		if tg.Account == account {
			return &tg, nil
		}
	}
	return nil, errors.New("not found")
}

// GetAccountOriginalRole
func GetAccountOriginalRole(account common.Address, number uint64) (common.RoleType, error) {
	val, err := Ide.ldb.Get(big.NewInt(int64(number)).Bytes(), nil)
	if err != nil {
		return common.RoleNil, err
	}

	es := new(mc.TopologyGraph)
	err = json.Unmarshal(val, &es)
	if err != nil {
		return common.RoleNil, err
	}

	for _, tg := range es.NodeList {
		if tg.Account == account {
			return tg.Type, nil
		}
	}
	return common.RoleNil, errors.New("not found")
}

//TODO:stub
func (ide *Identity) Node2CommonAddr(nodeid string) common.Address {
	value, ok := node2account[nodeid]
	if ok {
		return common.HexToAddress(value)
	}
	log.Error("ca", "wrong ndoe id ", nodeid)
	return common.Address{0}
}
