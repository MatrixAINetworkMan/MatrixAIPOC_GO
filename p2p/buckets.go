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
package p2p

import (
	"container/ring"
	"math/big"
	"math/rand"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/ca"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/mc"
	"github.com/ethereum/go-ethereum/p2p/discover"
)

// hash bucket
type Bucket struct {
	role   common.RoleType
	bucket map[int64][]discover.NodeID

	rings *ring.Ring
	lock  *sync.RWMutex

	ids []discover.NodeID

	sub event.Subscription

	blockChain chan mc.BlockToBucket
	quit       chan struct{}

	log log.Logger
}

// Init bucket.
var Buckets = &Bucket{
	role:  common.RoleNil,
	lock:  new(sync.RWMutex),
	ids:   make([]discover.NodeID, 0),
	quit:  make(chan struct{}),
	rings: ring.New(4),
}

const (
	MaxLink = 3
)

// init bucket.
func (b *Bucket) init() {
	for i := 0; i < b.rings.Len(); i++ {
		b.rings.Value = int64(i)
		b.rings = b.rings.Next()
	}
	b.log = log.New()
}

// Start bucket.
func (b *Bucket) Start() {
	b.init()

	b.log.Info("buckets start!")

	defer func() {
		b.log.Info("buckets stop!")
		b.sub.Unsubscribe()

		close(b.quit)
		close(b.blockChain)
	}()

	b.blockChain = make(chan mc.BlockToBucket)
	b.sub, _ = mc.SubscribeEvent(mc.BlockToBuckets, b.blockChain)

	for {
		select {
		case h := <-b.blockChain:
			// only bottom nodes will into this buckets.
			if h.Role > common.RoleBucket {
				continue
			}
			b.role = h.Role
			b.log.Info("bucket", "receive msg:", h.Ms)
			b.log.Info("bucket", "role", b.role)

			b.ids = h.Ms
			// maintain nodes in buckets
			b.maintainNodes(h.Ms)

			// adjust bucket order
			temp := &big.Int{}
			if temp.Mod(h.Height, big.NewInt(300)) == big.NewInt(50) {
				b.rings = b.rings.Prev()
			}
			b.log.Info("bucket", "ids", b.ids)
			b.log.Info("bucket", "bucket:", b.bucket)

			if len(b.ids) <= 64 {
				b.maintainOuter()
				continue
			}
			// if not in bucket, do nothing
			if b.role != common.RoleBucket {
				b.linkBucketPeer()
				continue
			}
			// maintain inner
			b.maintainInner()

			// maintain outer
			switch b.selfBucket() {
			case b.rings.Value:
				b.maintainOuter()
			case b.rings.Next().Value:
				b.disconnectMiner()
			case b.rings.Prev().Value:
				b.outer(MaxLink)
			default:
			}
		case <-b.quit:
			return
		}
	}
}

// Stop bucket running.
func (b *Bucket) Stop() {
	b.lock.Lock()
	b.quit <- struct{}{}
	b.lock.Unlock()
}

// maintainNodes maintain nodes in buckets.
func (b *Bucket) maintainNodes(elected []discover.NodeID) {
	b.bucket = make(map[int64][]discover.NodeID)
	for _, v := range elected {
		b.bucketAdd(v)
	}
}

// DisconnectMiner older disconnect miner.
func (b *Bucket) disconnectMiner() {
	miners := ca.Ide.GetRoleGroup(common.RoleMiner)
	for _, peer := range ServerP2p.Peers() {
		for _, miner := range miners {
			if peer.ID() == miner {
				peer.Disconnect(DiscQuitting)
				break
			}
		}
	}
}

// MaintainInner maintain bucket inner.
func (b *Bucket) maintainInner() {
	if len(b.ids) <= 64 {
		b.maintainOuter()
	}
	count := 0
	for _, peer := range ServerP2p.Peers() {
		if b.peerBucket(peer.ID()) == b.rings.Next().Value {
			count++
		}
	}
	if count < MaxLink {
		if MaxLink < len(b.bucket[b.rings.Next().Value.(int64)]) {
			b.inner(MaxLink-count, b.rings.Next().Value.(int64))
			return
		}
		b.inner(len(b.bucket[b.rings.Next().Value.(int64)])-count, b.rings.Next().Value.(int64))
	}
}

// MaintainOuter maintain bucket outer.
func (b *Bucket) maintainOuter() {
	count := 0
	miners := ca.Ide.GetRoleGroup(common.RoleMiner)
	b.log.Info("peer info", "p2p", miners)
	for _, peer := range ServerP2p.Peers() {
		for _, miner := range miners {
			if peer.ID() == miner {
				count++
				break
			}
		}
	}
	b.log.Info("peer count", "p2p", count)
	if count < MaxLink {
		if MaxLink < len(miners) {
			b.outer(MaxLink - count)
			return
		}
		b.outer(len(miners) - count)
	}
}

// SelfBucket return self bucket number.
func (b *Bucket) selfBucket() *big.Int {
	return b.peerBucket(ServerP2p.Self().ID)
}

func (b *Bucket) peerBucket(node discover.NodeID) *big.Int {
	key, _ := node.Pubkey()
	addr := crypto.PubkeyToAddress(*key)
	m := big.Int{}
	return m.Mod(addr.Hash().Big(), big.NewInt(4))
}

func (b *Bucket) deleteIdsById(nodeId discover.NodeID) {
	if len(b.ids) <= 0 {
		return
	}
	temp := 0
	for index, node := range b.ids {
		if node == nodeId {
			temp = index
			break
		}
	}
	b.ids = append(b.ids[:temp], b.ids[temp+1:]...)
}

func (b *Bucket) linkBucketPeer() {
	self := b.selfBucket().Int64()
	count := 0
	for _, peer := range ServerP2p.Peers() {
		if b.peerBucket(peer.ID()).Int64() == self {
			count++
		}
	}

	if count < MaxLink {
		if MaxLink < len(b.bucket[self]) {
			b.inner(MaxLink-count, b.rings.Value.(int64))
			return
		}
		b.inner(len(b.bucket[self])-count, b.rings.Value.(int64))
	}
}

// BucketAdd add to bucket.
func (b *Bucket) bucketAdd(nodeId discover.NodeID) {
	b.lock.Lock()

	key, _ := nodeId.Pubkey()
	addr := crypto.PubkeyToAddress(*key)
	m := big.Int{}
	mod := m.Mod(addr.Hash().Big(), big.NewInt(4)).Int64()

	b.bucket[mod] = append(b.bucket[mod], nodeId)
	b.lock.Unlock()
}

// BucketDel delete from bucket.
func (b *Bucket) bucketDel(nodeId discover.NodeID) {
	b.lock.Lock()
	defer b.lock.Unlock()

	key, _ := nodeId.Pubkey()
	addr := crypto.PubkeyToAddress(*key)
	m := big.Int{}
	mod := m.Mod(addr.Hash().Big(), big.NewInt(4)).Int64()

	if len(b.bucket[mod]) <= 0 {
		return
	}
	temp := 0
	for index, node := range b.bucket[mod] {
		if node == nodeId {
			temp = index
			break
		}
	}
	b.bucket[mod] = append(b.bucket[mod][:temp], b.bucket[mod][temp+1:]...)
}

// RandomPeers random peers from next buckets.
func (b *Bucket) randomInnerPeersByBucketNumber(num int, bucket int64) (nodes []discover.NodeID) {
	length := len(b.bucket[b.rings.Next().Value.(int64)])

	if length <= 0 {
		return nil
	}
	if length <= 3 {
		return b.bucket[bucket]
	}

	randoms := random(length, num)
	for _, ran := range randoms {
		for index := range b.bucket[bucket] {
			if index == ran {
				nodes = append(nodes, b.bucket[bucket][index])
				break
			}
		}
	}
	return nodes
}

// RandomOuterPeers random peers from overstory.
func (b *Bucket) randomOuterPeers(num int) (nodes []discover.NodeID) {
	minerNodes := ca.Ide.GetRoleGroup(common.RoleMiner)

	if len(minerNodes) <= 0 {
		return nil
	}
	if len(minerNodes) <= 3 {
		return minerNodes
	}

	randoms := random(len(minerNodes), num)
	for _, ran := range randoms {
		for index := range minerNodes {
			if ran == index {
				nodes = append(nodes, minerNodes[index])
			}
		}
	}
	return nodes
}

// inner adjust inner network.
func (b *Bucket) inner(num int, bucket int64) {
	if num <= 0 {
		return
	}
	peers := b.randomInnerPeersByBucketNumber(num, bucket)

	for _, value := range peers {
		b.log.Info("peer", "p2p", value)
		node := ServerP2p.ntab.Resolve(value)
		if node != nil {
			b.log.Info("buckets nodes", "p2p", node.ID)
			ServerP2p.AddPeer(node)
		}
	}
}

// outer adjust outer network.
func (b *Bucket) outer(num int) {
	if num <= 0 {
		return
	}
	peers := b.randomOuterPeers(num)

	for _, value := range peers {
		b.log.Info("peer", "p2p", value)
		node := ServerP2p.ntab.Resolve(value)
		if node != nil {
			b.log.Info("buckets nodes", "p2p", node.ID)
			ServerP2p.AddPeer(node)
		}
	}
}

// random a int number.
func random(i, num int) (randoms []int) {
	rand.Seed(time.Now().UnixNano())
	for m := 0; m < num; m++ {
		randoms = append(randoms, rand.Intn(i))
	}
	return randoms
}
