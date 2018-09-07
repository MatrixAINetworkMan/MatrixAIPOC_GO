package random

import (
	"github.com/ethereum/go-ethereum/mc"
)

const (
	ModuleSeed = "ElectionSeed"
	ModuleVote = "RandomVote"
)

type Random struct {
	electionseed *ElectionSeed
	randomvote   *RandomVote
}

func New(msgcenter *mc.Center) (*Random, error) {
	random := &Random{}
	var err error
	random.electionseed, err = newElectionSeed(msgcenter)
	if err != nil {
		return nil, err
	}
	random.randomvote, err = newRandomVote(msgcenter)
	if err != nil {
		return nil, err
	}

	return random, nil

}
