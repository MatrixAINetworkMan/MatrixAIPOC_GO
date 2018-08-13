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
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/discover"
)

// hash bucket
type Bucket struct {
	role   common.RoleType
	bucket map[*big.Int][]discover.NodeID

	rings *ring.Ring
	lock  *sync.RWMutex

	BlockChain chan blockChain
	quit       chan struct{}
}

type blockChain struct {
	role common.RoleType
	head interface{}
}

// Init bucket.
var Buckets = &Bucket{
	lock:       new(sync.RWMutex),
	bucket:     make(map[*big.Int][]discover.NodeID),
	quit:       make(chan struct{}),
	BlockChain: make(chan blockChain),
	rings:      ring.New(4),
}

const (
	MaxLink    = 3
	MaxContent = 2000
)

// init bucket.
func (b *Bucket) init() {
	for i := 0; i < b.rings.Len(); i++ {
		b.rings.Value = big.NewInt(int64(i))
		b.rings = b.rings.Next()
	}
}

// Start bucket.
func (b *Bucket) Start() {
	b.init()

	defer func() {
		close(b.quit)
		close(b.BlockChain)
	}()

	for {
		select {
		case h := <-b.BlockChain:
			b.role = h.role
			b.bucketAdd(discover.NodeID{})
			b.bucketDel(discover.NodeID{})

			// adjust bucket order
			header := h.head.(types.Header)
			temp := &big.Int{}
			if temp.Mod(header.Number, big.NewInt(300)) == big.NewInt(50) {
				b.rings = b.rings.Prev()
			}

			// if not in bucket, do nothing
			if b.role != common.RoleBucket {
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
	count := 0
	for _, peer := range ServerP2p.Peers() {
		if b.peerBucket(peer.ID()) == b.rings.Next().Value {
			count++
		}
	}
	if count < MaxLink {
		if MaxLink < len(b.bucket[b.rings.Next().Value.(*big.Int)]) {
			b.inner(MaxLink - count)
			return
		}
		b.inner(len(b.bucket[b.rings.Next().Value.(*big.Int)]) - count)
	}
}

// MaintainOuter maintain bucket outer.
func (b *Bucket) maintainOuter() {
	count := 0
	miners := ca.Ide.GetRoleGroup(common.RoleMiner)
	for _, peer := range ServerP2p.Peers() {
		for _, miner := range miners {
			if peer.ID() == miner {
				count++
				break
			}
		}
	}
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
	key, _ := ServerP2p.Self().ID.Pubkey()
	addr := crypto.PubkeyToAddress(*key)
	m := big.Int{}
	return m.Mod(addr.Hash().Big(), big.NewInt(4))
}

func (b *Bucket) peerBucket(node discover.NodeID) *big.Int {
	key, _ := node.Pubkey()
	addr := crypto.PubkeyToAddress(*key)
	m := big.Int{}
	return m.Mod(addr.Hash().Big(), big.NewInt(4))
}

// BucketAdd add to bucket.
func (b *Bucket) bucketAdd(nodeId discover.NodeID) {
	b.lock.Lock()

	key, _ := nodeId.Pubkey()
	addr := crypto.PubkeyToAddress(*key)
	m := big.Int{}
	mod := m.Mod(addr.Hash().Big(), big.NewInt(4))

	b.bucket[mod] = append(b.bucket[mod], nodeId)
	b.lock.Unlock()
}

// BucketDel delete from bucket.
func (b *Bucket) bucketDel(nodeId discover.NodeID) {
	b.lock.Lock()

	key, _ := nodeId.Pubkey()
	addr := crypto.PubkeyToAddress(*key)
	m := big.Int{}
	mod := m.Mod(addr.Hash().Big(), big.NewInt(4))

	temp := 0
	for index, node := range b.bucket[mod] {
		if node == nodeId {
			temp = index
			break
		}
	}
	b.bucket[mod] = append(b.bucket[mod][:temp], b.bucket[mod][temp+1:]...)
	b.lock.Unlock()
}

// RandomPeers random peers from next buckets.
func (b *Bucket) randomInnerPeers(num int) (nodes []discover.NodeID) {
	length := len(b.bucket[b.rings.Next().Value.(*big.Int)])

	if length <= 0 {
		return nil
	}
	if length <= 3 {
		return b.bucket[b.rings.Next().Value.(*big.Int)]
	}

	randoms := random(length, num)
	for _, ran := range randoms {
		for index := range b.bucket[b.rings.Next().Value.(*big.Int)] {
			if index == ran {
				nodes = append(nodes, b.bucket[b.rings.Next().Value.(*big.Int)][index])
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
func (b *Bucket) inner(num int) {
	if num <= 0 {
		return
	}

	peers := b.randomInnerPeers(num)
	for index := range peers {
		go ServerP2p.ntab.Lookup(peers[index])
	}
}

// outer adjust outer network.
func (b *Bucket) outer(num int) {
	if num <= 0 {
		return
	}

	peers := b.randomOuterPeers(num)
	for index := range peers {
		go ServerP2p.ntab.Lookup(peers[index])
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
