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
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

const (
	PreBlock uint64 = 1
	ProBlock        = 2

	BootNodeD string = "f26fa4112f2cc603a114d3eec20d5a4605debe1c3cecc36c347982aaf3c30e5790b9f936c2c4e6862e615255cb1bce05a1578f4e9766ab43991c87864d3ff1fe"
	BootNodeV string = "0e6376b1409eb0d0308edc685e71536e246fe3264295d276be2b953998f43e8c8c70907a0580f7437f2fb249fcc6d30f9ba24180fa6b6e499f61646268f54517"
	BootNodeM string = "88e3f601edac6f553ec5d65ed5e543e858da1590ae3ad922f3e077189823b4d1b24e5028de6424ed59ffe76e88e52811ac879e7f5b3b1bc45e71090811653168"
	BootNodes string = "14055bc410ec72424c38a25b05ad1ff711eb230c434d16c20da8f2ef7ee73ec9dd32aabe87c1003003b717fc45354cbca0006c9d997e206b01326c9faeeba365"

	MaxId uint = 256
)

const (
	BroadcastInterval     = 100 //broadcast interval
	ReelectionInterval    = 300 //re-election interval
	VerifyNetChangeUpTime = 40  //validator network change time (upfront time)
	MinerNetChangeUpTime  = 30  //miner network change time (upfront time)
)

type TopologyNodeInfo struct {
	Account  common.Address
	Position uint16
	Type     common.RoleType
	Stock    uint16
}

type TopologyGraph struct {
	Number   *big.Int
	NodeList []TopologyNodeInfo
}
