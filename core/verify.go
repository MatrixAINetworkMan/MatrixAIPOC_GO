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
// Boot_Find project Boot_Find.go
package core

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/rlp"
)

const (
	followerNum int = 10
)

type VoteResultSigner interface {
	/* TODO: add methods */
	Sender(vr *voteResult) (common.Address, error)
	SignatureValues(vr *voteResult, sig []byte) (r, s, v *big.Int, err error)
	Hash(vr *voteResult) common.Hash
	Equal(VoteResultSigner) bool
}

type voteResult struct {
	invalidTx []bool

	fee int //DEPOSITS

	//Signature values
	V *big.Int
	R *big.Int
	S *big.Int
}

func rlpHash(x interface{}) (h common.Hash) {
	hw := sha3.NewKeccak256()
	rlp.Encode(hw, x)
	hw.Sum(h[:0])
	return h
}

func recoverPlain(sighash common.Hash, R, S, Vb *big.Int, homestead bool) (common.Address, error) {
	if Vb.BitLen() > 8 {
		return common.Address{}, types.ErrInvalidSig
	}
	V := byte(Vb.Uint64() - 27)
	if !crypto.ValidateSignatureValues(V, R, S, homestead) {
		return common.Address{}, types.ErrInvalidSig
	}
	// encode the snature in uncompressed format
	r, s := R.Bytes(), S.Bytes()
	sig := make([]byte, 65)
	copy(sig[32-len(r):32], r)
	copy(sig[64-len(s):64], s)
	sig[64] = V
	// recover the public key from the snature
	pub, err := crypto.Ecrecover(sighash[:], sig)
	if err != nil {
		return common.Address{}, err
	}
	if len(pub) == 0 || pub[0] != 4 {
		return common.Address{}, errors.New("invalid public key")
	}
	var addr common.Address
	copy(addr[:], crypto.Keccak256(pub[1:])[12:])
	return addr, nil
}

type LeaderVoteResultSigner struct{}

func (l LeaderVoteResultSigner) Equal(s2 VoteResultSigner) bool {
	_, ok := s2.(LeaderVoteResultSigner)
	return ok
}

func (l LeaderVoteResultSigner) Hash(vr *voteResult) common.Hash {
	return rlpHash([]interface{}{
		vr.invalidTx,
	})
}

func (l LeaderVoteResultSigner) Sender(vr *voteResult) (common.Address, error) {
	return recoverPlain(l.Hash(vr), vr.R, vr.S, vr.V, false)

}

func (l LeaderVoteResultSigner) SignatureValues(vr *voteResult, sig []byte) (r, s, v *big.Int, err error) {
	if len(sig) != 65 {
		panic(fmt.Sprintf("Wrong size of signature: got %d, want 65", len(sig)))
	}
	r = new(big.Int).SetBytes(sig[:32])
	s = new(big.Int).SetBytes(sig[32:64])
	v = new(big.Int).SetBytes([]byte{sig[64] + 27})
	return r, s, v, nil
}

func (vr *voteResult) WithSignature(signer LeaderVoteResultSigner, sig []byte) (*voteResult, error) {
	r, s, v, err := signer.SignatureValues(vr, sig)
	if err != nil {
		return nil, err
	}
	vr.R, vr.S, vr.V = r, s, v
	return vr, nil
}

func SignVoteResult(v *voteResult, s LeaderVoteResultSigner, prv *ecdsa.PrivateKey) (*voteResult, error) {
	h := s.Hash(v)
	sig, err := crypto.Sign(h[:], prv)
	if err != nil {
		return nil, err
	}
	return v.WithSignature(s, sig)
}

func PackageTxInPool(pool *TxPool) []types.Transaction {

	txs := make([]types.Transaction, 0)

	localTxs,_ := pool.Pending()

	for _, v := range localTxs {
		for _, tx := range v {
			txs = append(txs, *tx)
		}
	}

	return txs
}

func ValidateLocalTx(pool *TxPool, tx *types.Transaction) error {

	return pool.validateTx(tx, false)
}
