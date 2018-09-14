package broadcastTx

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/mc"
	"sync"
)

const (
	// chainHeadChanSize is the size of channel listening to ChainHeadEvent.
	sendBroadCastCHSize = 10
)

type BroadCast struct {
	ethBackend ethapi.Backend

	sendBroadCastCH chan mc.BroadCastEvent
	broadCastSub    event.Subscription
	wg              sync.WaitGroup
}

func NewBroadCast(apiBackEnd ethapi.Backend) *BroadCast {

	bc := &BroadCast{
		ethBackend:      apiBackEnd,
		sendBroadCastCH: make(chan mc.BroadCastEvent, sendBroadCastCHSize),
	}
	bc.broadCastSub, _ = mc.SubscribeEvent(mc.SendBroadCastTx, bc.sendBroadCastCH)
	bc.wg.Add(1)
	go bc.loop()
	return bc
}
func (bc *BroadCast) Start() {
	//go bc.loop()
}
func (bc *BroadCast) Stop() {
	bc.broadCastSub.Unsubscribe()
	bc.wg.Wait()
	log.Info("BroadCast Server stopped.--YY")
}

func (bc *BroadCast) loop() {
	defer bc.wg.Done()
	for {
		select {
		case ev := <-bc.sendBroadCastCH:
			bc.sendBroadCastTransaction(ev.Txtyps, ev.Height, ev.Data)
		case <-bc.broadCastSub.Err():
			return
		}
	}
}

//YY Interface of Broadcast Transactions
func (bc *BroadCast) sendBroadCastTransaction(t string, h *big.Int, data []byte) error {
	if h.Cmp(big.NewInt(100)) < 0 {
		log.Info("===Send BroadCastTx===", "block height less than 100")
		return errors.New("===Send BroadCastTx===,block height less than 100")
	}
	log.Info("=========YY=========", "sendBroadCastTransaction", data)
	bType := false
	if t == mc.CallTheRoll {
		bType = true
	}
	h.Quo(h, big.NewInt(100))
	t += h.String()
	tmpData := make(map[string][]byte)
	tmpData[t] = data
	msData, _ := json.Marshal(tmpData)
	var txtype byte
	txtype = byte(1)
	tx := types.NewHeartTransaction(txtype, msData)
	var chainID *big.Int
	if config := bc.ethBackend.ChainConfig(); config.IsEIP155(bc.ethBackend.CurrentBlock().Number()) {
		chainID = config.ChainId
	}
	signed, err := bc.ethBackend.SignTx(tx, chainID)
	if err != nil {
		log.Info("=========YY=========", "sendBroadCastTransaction:SignTx=", err)
		return err
	}
	err1 := bc.ethBackend.SendBroadTx(context.Background(), signed, bType)
	log.Info("=========YY=========", "sendBroadCastTransaction:Return=", err1)
	return nil
}
