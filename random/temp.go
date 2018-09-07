package random

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

var Prv *big.Int
var Pub []byte
var Address *big.Int
var Address1 common.Address

func asd() {

	fmt.Println("AAAAAAAAAAAAAAAAAAAAAAAA")
	if Prv == nil {
		Prv, Pub, _ = getkey()
		log.Info("asd", "a", Prv, "as", Pub, "asdsa", Address1)
	}

	Address = big.NewInt(100)
	Address1 = common.BigToAddress(Address)
	log.Info("asd", "a", Prv, "as", Pub, "asdsa", Address1)

}

func GetKeyTransInfo(Heigh uint64, types string) map[common.Address][]byte {
	asd()
	mmap_private := make(map[common.Address][]byte)
	mmap_public := make(map[common.Address][]byte)

	mmap_private[Address1] = Prv.Bytes()
	mmap_public[Address1] = Pub

	fmt.Println("GGGGGGGGGGGGG", Prv, Prv.Bytes(), Pub)

	/*
		fmt.Println(Address1, mmap_private, mmap_public)

		private1 := mmap_private[Address1]
		public1 := mmap_public[Address1]
		if compare(public1, private1) == true {
			fmt.Println("biggo")
		} else {
			fmt.Println("sb")
		}*/
	if types == "private" {
		return mmap_private
	}
	return mmap_public
}
