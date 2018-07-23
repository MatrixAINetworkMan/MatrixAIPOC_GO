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
package verifier

import (
	"encoding/json"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/election"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
)

const (
	modulName string = "VERIFIER"
	nodeBusy         = iota
	nodeIdle
)

const (
	TIMEUNKNOW = iota
	TIMEOUT    //timeout
	TIMEIN     //finish
)

type Verifier struct {
	role   uint8 //1,follower; 2: leader
	chain  *core.BlockChain
	txPool *core.TxPool

	leaderProcessOnceCh chan uint64
	followProcessOnceCh chan uint64
	processStopCh       chan struct{}
	notifyChan          chan uint64
	quitChan            chan struct{}

	leaderTxRecvCh chan []types.Transaction // channel of leader recv txs
	leaderWorkCh   chan uint8               // channel of leader recv timeout or timein msg
	invalidTxChan  chan []bool              // channel of leader recv invalid tx list
	voteResultChan chan VoteResult

	followerVoteReqCh chan []uint16
	followerTxRecvCh  chan []types.Transaction // channel of follower recv tx
	followerMsgCh     chan int                 // channel of follower recv msg

	minerList              []election.NodeInfo
	verifierList           []election.NodeInfo
	localNodeInfo          election.NodeInfo
	leadNodeInfo           election.NodeInfo
	follower               []string
	sessionPM              sessionType
	nodeState              uint8
	lock                   sync.RWMutex
	invalidTxsList         []bool
	leaderRemoveTxList     []uint16
	followerInvalidTxsList []uint16
}

func New(chain *core.BlockChain, pool *core.TxPool) *Verifier {
	verifier := &Verifier{
		chain:               chain,
		txPool:              pool,
		leaderProcessOnceCh: make(chan uint64, 1),
		leaderWorkCh:        make(chan uint8, 1),
		followProcessOnceCh: make(chan uint64),
		processStopCh:       make(chan struct{}),
		notifyChan:          make(chan uint64),
		quitChan:            make(chan struct{}),
		leaderTxRecvCh:      make(chan []types.Transaction),
		invalidTxChan:       make(chan []bool),
		voteResultChan:      make(chan VoteResult),
		followerTxRecvCh:    make(chan []types.Transaction),
		followerVoteReqCh:   make(chan []uint16),
		followerMsgCh:       make(chan int),
		nodeState:           nodeIdle,
		sessionPM:           sessionType{sessionState: sessionIdle},
	}

	sp2p := p2p.Slove_Cycle{}
	sp2p.Register_verifier(verifier.receiveP2PMsg, nil)

	go verifier.waitForScheduler()
	go verifier.leaderNewProcess()
	go verifier.followerProcess()
	return verifier
}

func (v *Verifier) waitForScheduler() {
	for {
		select {
		case blockNum := <-v.notifyChan:
			log.Info(modulName, "Rcv Block Event", blockNum)
			if v.nodeState == nodeBusy {
				log.Info(modulName, "ENTER VERIFIER", blockNum)

				//通过nodelist和nodeid计算出当前验证者是leader还是follower
				v.electionLeader(v.verifierList, blockNum)
				log.Info(modulName, "verifierRole", v.role)
				switch v.role {
				case leader:
					readySendNodelist(v, blockNum)
					log.Info(modulName, "leader", "")
					if v.sessionPM.sessionState == sessionIdle {
						v.sessionPM.reset()
						v.sessionPM.sessionState = sessionTxmsg1
						v.leaderWorkCh <- TIMEUNKNOW

					} else {
						log.Info(modulName, "Block To Fast", "")
					}
				case follower:
					log.Info(modulName, "This is follower node!", "")
					//no need to do anything
				default:
					log.Info(modulName, "invalid role", v.role)
				}
			}

		case <-v.quitChan:
			return
		}
	}
}

var txs []types.Transaction
var validTx []types.Transaction = make([]types.Transaction, 0)
var vrlist []VoteResult

func (v *Verifier) sendMsg1() {

	//clear vote list
	vrlist = make([]VoteResult, 0)

	//package leader's transactions
	validTx = make([]types.Transaction, 0)
	txs = core.PackageTxInPool(v.txPool)
	validTx = txs
	//send transaction request to follower
	v.sendTxRequestToFollowers()
	v.sessionPM.updatestate(sessionRxmsg2)
}

func (v *Verifier) sendMsg3() {

	log.Info(modulName, "Leader Session, Rx Msg2", len(v.sessionPM.rxMsg2List), "node", v.sessionPM.rxMsg2List)
	//tx msg3, rcv msg4
	txs = core.PackageTxInPool(v.txPool)
	v.sendTxToFollower(txs)
	v.sessionPM.updatestate(sessionRxmsg4)
	log.Info(modulName, "Leader Session , Tx Msg3", len(txs))
}

func (v *Verifier) sendMsg5() {
	log.Info(modulName, "Leader Session, Rx Msg4", len(v.sessionPM.rxMsg4List), "node", v.sessionPM.rxMsg4List)
	log.Info(modulName, "Leader Session, Rx Msg4,  valid trans num", len(validTx))
	log.Info(modulName, "Leader Session, Rx Msg4,  valid trans ", validTx)
	//TODO:leader voteToTx()
	//tx msg5, rcv msg6
	v.sendVoteRequestToFollower()
	v.sessionPM.updatestate(sessionRxmsg6)
}

func (v *Verifier) leaderVoteToTx(leaderInvalidTxList []uint16) {

	var vr VoteResult
	vr.NodeId = v.localNodeInfo.ID
	vr.Result = true
	vrlist = append(vrlist, vr)
	log.Info(modulName, "leader session Vote to Transaction result", vrlist)
}

func (v *Verifier) makeDPOS() {

	log.Info(modulName, "Leader Session, Rx Msg6", len(v.sessionPM.rxMsg6List), "node", v.sessionPM.rxMsg6List)
	voteLen := len(vrlist)
	log.Info(modulName, "Leader Session, Rx Msg6, vote num", voteLen)
	log.Info(modulName, "Leader Session, Rx Msg6, vrlist", vrlist)

	if ret := v.DposTx(vrlist); ret {
		var msg ConsesusResult
		msg.Txs = validTx
		log.Info(modulName, "Leader Session, vot,  valid trans ", validTx)
		msg.Result = vrlist
		data, err := json.MarshalIndent(msg, "", "   ")
		if err != nil {
			log.Info(modulName, "to miner", err)
		}
		v.sendMsgToMiner(data, Transaction, blockNum)
		log.Info(modulName, "leader Session", "", "Tx Miner trans Num", len(vrlist))
	} else {
		log.Info(modulName, "Leader Session,  vote fail")
	}
	v.sessionPM.updatestate(sessionIdle)
}

func (v *Verifier) leaderNewRun() {
	switch v.sessionPM.sessionState {
	case sessionIdle:
	case sessionTxmsg1:
		v.sendMsg1()
	case sessionRxmsg2:
	case sessionTxmsg3:
		v.sendMsg3()
	case sessionRxmsg4:
	case sessionTxmsg5:
		v.sendMsg5()
	case sessionRxmsg6:

	case sessionDpos:
		leaderVote := VoteResult{true, v.localNodeInfo.ID}
		vrlist = append(vrlist, leaderVote)
		v.makeDPOS()
		v.sessionPM.updatestate(sessionIdle)
	}
}

func (v *Verifier) timeout(status *uint8, timeout time.Duration) {
	time.Sleep(timeout)
	if TIMEUNKNOW == *status {
		v.leaderWorkCh <- TIMEOUT
	} else {
		return
	}
}

func (v *Verifier) leaderNewProcess() {
	var pretimeout *uint8
	log.Info(modulName, "enter leaderNewProcess", "")
	for {
		select {
		case status := <-v.leaderWorkCh:
			timeout := new(uint8)
			log.Info(modulName, "leaderWorkCh", "", "status", status)

			*timeout = TIMEUNKNOW
			//handle timeout or finish
			if TIMEOUT == status {
				log.Info(modulName, "pre timeout:", "")
				if v.sessionPM.rxMsgCount == 0 {
					// no followers alive
					v.sessionPM.updatestate(sessionDpos)
				} else {
					v.sessionPM.rxMsgCount = 0
					v.sessionPM.sessionState++
				}
			} else if TIMEIN == status {
				*pretimeout = TIMEIN //改变前置
				v.sessionPM.rxMsgCount = 0
				log.Info(modulName, "pre finish:", "")
				v.sessionPM.sessionState++
			}

			v.leaderNewRun()
			if v.sessionPM.sessionState > sessionTxmsg1 && v.sessionPM.sessionState < sessionDpos {

				log.Info(modulName, "sessionState", v.sessionPM.sessionState)
				//go v.timeout(timeout, 50*time.Microsecond)
				go v.timeout(timeout, 1*time.Second)

			}
			pretimeout = timeout
		}
	}

	return
}

func (v *Verifier) followerProcess() {
	for {
		select {
		case <-v.followerMsgCh:
			txs := core.PackageTxInPool(v.txPool)
			log.Info(modulName, "Follower Rcv msg1", "", "ack msg2, txs len", len(txs))
			log.Info(modulName, "Follower Rcv msg1", "", "ack msg2, txs ", txs)
			v.followerInvalidTxsList = make([]uint16, 0)
			go v.sendTxToLeader(txs)

		case leaderTx := <-v.followerTxRecvCh:
			//验证交易
			invalidTx := make([]uint16, 0)
			for k, tx := range leaderTx {
				// If the transaction fails basic validation, discard it
				if err := core.ValidateLocalTx(v.txPool, &tx); err != nil {
					invalidTx = append(invalidTx, uint16(k))
				}
			}
			v.followerInvalidTxsList = invalidTx
			log.Info(modulName, "Follower Rcv msg3", "", "ack msg4", "")
			go v.sendInvalidTxToLeader(invalidTx)

		case remTxList := <-v.followerVoteReqCh:
			//go v.voteToLeader(remTxList)
			log.Info(modulName, "Follower Rcv msg5", remTxList, "ack msg6", "")
			go v.voteToTx(remTxList)

		case <-v.processStopCh:
			return
		}
	}
}

func (v *Verifier) receiveP2PMsg(temp interface{}, data p2p.Custsend) {
	if v.nodeState == nodeIdle {
		return
	}

	switch v.role {
	case leader:
		go v.leaderProcessMsg(temp, data)
	case follower:
		go v.followerProcessMsg(temp, data)
	}
}

const (
	sessionIdle   = iota
	sessionTxmsg1 //send tx request to follower
	sessionRxmsg2 // recv transactions from follower
	sessionTxmsg3 // send transactions to follower
	sessionRxmsg4 // recv invalid transactions from follower
	sessionTxmsg5 // send vote request to follower
	sessionRxmsg6 // recv vote result frome follower
	sessionDpos
)

type sessionType struct {
	sessionState uint8
	rxMsgCount   int
	rxMsg2List   []string
	rxMsg4List   []string
	rxMsg6List   []string
}

func (s *sessionType) reset() {
	s.sessionState = sessionIdle
	s.rxMsg2List = make([]string, 0)
	s.rxMsg4List = make([]string, 0)
	s.rxMsg6List = make([]string, 0)
}
func (s *sessionType) updatestate(state uint8) {
	s.sessionState = state
}

func removeRepByMap(slc []uint16) []uint16 {
	result := []uint16{}
	tempMap := map[uint16]byte{} // 存放不重复主键
	for _, e := range slc {
		l := len(tempMap)
		tempMap[e] = 0
		if len(tempMap) != l { // 加入map后，map长度变化，则元素不重复
			result = append(result, e)
		}
	}
	return result
}
func (v *Verifier) leaderProcessMsg(temp interface{}, data p2p.Custsend) {
	log.Info(modulName, "P2P TO Leader Msg", "")
	if !v.msgFromVeifierNodeId(data.NodeId) {
		log.Info(modulName, "msg src incorrect, ID", data.NodeId, "IP", data.FromIp)
		return
	}
	switch data.Data.Type {
	case msgSendTxToLeader:
		log.Info(modulName, "Leader session, rx msg2", "node id", data.NodeId, "node ip", data.FromIp)
		if v.sessionPM.sessionState == sessionRxmsg2 {
			var txs []types.Transaction
			if err := json.Unmarshal(data.Data.Data_struct, &txs); err != nil {
				log.Info(modulName, "Leader session rx msg2, 反序列化高度信息失败")
			} else {
				log.Info(modulName, "Leader session  rx msg2, msg ", txs)
				v.sessionPM.rxMsg2List = append(v.sessionPM.rxMsg2List, data.FromIp)
				log.Info(modulName, "Leader session  rx msg2, msg num", len(txs))
				for _, tx := range txs {
					err := v.txPool.AddRemote(&tx)
					log.Info(modulName, "Leader session rx msg2, Add Txs", err)
				}
			}
			v.sessionPM.rxMsgCount++

			log.Info(modulName, "Leader session  rx msg2", "rxMsgCount", v.sessionPM.rxMsgCount)
			if (len(v.verifierList) - 1) == v.sessionPM.rxMsgCount {
				v.leaderWorkCh <- TIMEIN
			}
		} else {
			log.Info(modulName, "Leader session, rx msg2 ,timeout", "", "node ip", data.FromIp)
		}
	case msgSendInvalidTxToLeader:
		log.Info(modulName, "Leader session, rx msg4", "node id", data.NodeId, "node ip", data.FromIp)
		if v.sessionPM.sessionState == sessionRxmsg4 {
			var invalidTx []uint16
			if err := json.Unmarshal(data.Data.Data_struct, &invalidTx); err != nil {
				log.Info(modulName, "Leader session rx msg4, 反序列化高度信息失败")
			} else {
				v.sessionPM.rxMsg4List = append(v.sessionPM.rxMsg4List, data.NodeId)
				log.Info(modulName, "Leader session  rx msg4, invalidTx", invalidTx)
				//
				for i := 0; i < len(invalidTx); i++ {
					v.leaderRemoveTxList = append(v.leaderRemoveTxList, invalidTx[i])
				}
				v.leaderRemoveTxList = removeRepByMap(v.leaderRemoveTxList)
				log.Info(modulName, "Leader session  rx msg4", v.leaderRemoveTxList)
			}
			v.sessionPM.rxMsgCount++
			//接收完成
			log.Info(modulName, "Leader session  rx msg4", "rxMsgCount", v.sessionPM.rxMsgCount)
			if len(v.sessionPM.rxMsg2List) == v.sessionPM.rxMsgCount {
				v.leaderWorkCh <- TIMEIN
			}
		} else {
			log.Info(modulName, "Leader session, rx msg4 ,timeout", "", "node ip", data.FromIp)
		}

	case msgSendVoteResultToLeader:
		log.Info(modulName, "Leader session, rx msg6", "node id", data.NodeId, "node ip", data.FromIp)
		if v.sessionPM.sessionState == sessionRxmsg6 {
			var vr VoteResult
			if err := json.Unmarshal(data.Data.Data_struct, &vr); err != nil {
				log.Info(modulName, "Leader session rx msg6, 反序列化高度信息失败")
			} else {
				log.Info(modulName, "msg6 data", data.Data.Data_struct)

				v.sessionPM.rxMsg6List = append(v.sessionPM.rxMsg6List, data.NodeId)
				if !v.msgFromVeifierNodeId(vr.NodeId) {
					log.Info(modulName, "vote nodeid invalid", vr)
				}
				vrlist = append(vrlist, vr)
				log.Info(modulName, "Leader session  rx msg6", vrlist)
			}
		} else {
			log.Info(modulName, "Leader session, rx msg6 ,timeout", "node ip", data.FromIp)
		}

		v.sessionPM.rxMsgCount++
		log.Info(modulName, "Leader session  rx msg6", "rxMsgCount", v.sessionPM.rxMsgCount)
		if len(v.sessionPM.rxMsg4List) == v.sessionPM.rxMsgCount {
			v.leaderWorkCh <- TIMEIN
		}
	}
}

func (v *Verifier) followerProcessMsg(temp interface{}, data p2p.Custsend) {
	log.Info(modulName, "P2P TO Follower Msg", "", "Type", data.Data.Type)
	if !v.msgFromLeaderByNodeId(data.NodeId) {
		log.Info(modulName, "msg src incorrect, ID", data.NodeId, "IP", data.FromIp)
		return
	}
	switch data.Data.Type {
	case msgSendTxReqToFollower:
		var msg int
		if err := json.Unmarshal(data.Data.Data_struct, &msg); err != nil {
			log.Info(modulName, "反序列化高度信息失败", "")
		}
		log.Info(modulName, "follower recv msg", msg)
		v.followerMsgCh <- msg
	case msgSendTxToFollower:
		var txs []types.Transaction
		if err := json.Unmarshal(data.Data.Data_struct, &txs); err != nil {
			log.Info(modulName, "反序列化高度信息失败", "")
		} else {

		}
		log.Info(modulName, "Followers Rcv Msg3 Trans", len(txs))
		v.followerTxRecvCh <- txs
	case msgSendVoteReqToFollower:
		var msg []uint16
		if err := json.Unmarshal(data.Data.Data_struct, &msg); err != nil {
			log.Info("反序列化高度信息失败")
		}
		log.Info(modulName, "Follower Rcv Msg5 ", data.Data.Type, "Data", msg)
		v.followerVoteReqCh <- msg
	default:
		log.Info(modulName, "Follower Rcv Error Type", data.Data.Type)
	}
}

type VoteResult struct {
	//State
	Result bool
	NodeId string
}
type VoteList []VoteResult

var blockNum uint64

func (v *Verifier) isFromVerifyList(nodeID string) (election.NodeInfo, bool) {
	for _, node := range v.verifierList {
		if node.ID == nodeID {
			return node, true
		}
	}
	return election.NodeInfo{}, false
}

func (v *Verifier) DposTx(results []VoteResult) bool {
	validNum := 0
	passNum := 0
	wealthValid := uint64(0)
	wealthTotal := uint64(0)
	for _, res := range results {
		if node, isIn := v.isFromVerifyList(res.NodeId); isIn {
			validNum++
			if res.Result {
				passNum++
				wealthValid += node.Wealth
			}
			wealthTotal += node.Wealth
		}
	}
	log.Info(modulName, "leader session DposTx", "", "passNum", passNum, "validNum", validNum, "blocknum", blockNum)
	//must over half
	if passNum*2 <= validNum {
		return false
	}
	log.Info(modulName, "leader session DposTx", "", "wealthValid", wealthValid, "wealthTotal", wealthTotal, "blocknum", blockNum)
	//wealthValid must over 75% of wealthTotal
	if wealthValid*4 <= 3*wealthTotal {
		return false
	}
	return true
}

type ConsesusResult struct {
	Txs    []types.Transaction `json:"txs"`
	Result []VoteResult        `json:"result"`
	Fee    uint                `json:"fee"`
}

type MsgToMiner struct {
	Data     []byte `json:"data"`
	BlockNum uint64 `json:"block_num"`
	MsgType  uint   `json:"msg_type"` //1:Transection; 2:Broadcast
}

const (
	msgStart int = iota
	msgRequestTx
)

const (
	verifierToMiner    uint64 = 0x04
	verifierToVerifier uint64 = 0x05
)

func sendMsg(sendTo uint64, msgType uint64, msgData interface{}, toIP string, proto bool) {

	var t_Send []p2p.Custsend
	var t_data_format p2p.Data_Format
	t := fillMsgHeader(sendTo, toIP, proto)
	t_data_format.Type = msgType
	t_data_format.Data_struct, _ = json.MarshalIndent(msgData, "", "   ")
	t.Data = t_data_format
	t_Send = append(t_Send, t)
	p2p.CustSend(t_Send)
	log.Info(modulName, "send data to", toIP)
}

func (v *Verifier) isVerifierLeaderByNodeId(noidInfo election.NodeInfo) bool {
	return noidInfo.ID == v.leadNodeInfo.ID
}
func (v *Verifier) msgFromLeaderByNodeId(nodeId string) bool {
	return nodeId == v.leadNodeInfo.ID
}
func (v *Verifier) msgFromVeifierNodeId(nodeId string) bool {
	for i := 0; i < len(v.verifierList); i++ {
		if nodeId == v.verifierList[i].ID {
			return true
		}
	}
	return false
}

func (v *Verifier) sendTxRequestToFollowers() {
	for i := 0; i < len(v.verifierList); i++ {
		if v.isVerifierLeaderByNodeId(v.verifierList[i]) {
			continue
		}
		log.Info(modulName, "Leader Session Send msgSendTxReqToFollower", "")
		sendMsg(verifierToVerifier, msgSendTxReqToFollower, msgRequestTx, v.verifierList[i].IP, false)
	}
	return
}

func (v *Verifier) sendTxToFollower(txs []types.Transaction) {
	for i := 0; i < len(v.verifierList); i++ {
		if v.isVerifierLeaderByNodeId(v.verifierList[i]) {
			continue
		}
		sendMsg(verifierToVerifier, msgSendTxToFollower, txs, v.verifierList[i].IP, false)
	}
	return
}

func fillMsgHeader(code uint64, toip string, istcp bool) p2p.Custsend {
	var t p2p.Custsend
	t.ToIp = toip
	t.IsTcp = istcp
	t.Code = code

	return t

}

const (
	verifierMsgType uint64 = iota
	msgSendTxReqToFollower
	msgSendTxToFollower
	msgSendVoteReqToFollower
	msgSendTxToLeader
	msgSendInvalidTxToLeader
	msgSendVoteResultToLeader
	sendTxsToMiner
)

const (
	MsgToMinerStart uint = iota
	Transaction
	Broadcast
)

func (v *Verifier) sendMsgToMiner(data []byte, msgType uint, blockNum uint64) {
	var msg MsgToMiner
	msg.Data = data
	msg.BlockNum = blockNum
	msg.MsgType = msgType
	log.Info("send msg to miner", "data", string(msg.Data), "type", msg.MsgType)
	v.sendToMiner(msg)
	return
}

func (v *Verifier) sendToMiner(msg MsgToMiner) {
	for i := 0; i < len(v.minerList); i++ {
		go sendMsg(verifierToMiner, sendTxsToMiner, msg, v.minerList[i].IP, false)
	}
}

func (v *Verifier) sendVoteRequestToFollower() {
	log.Info(modulName, "tx msg 5, listLen", len(v.verifierList))
	for i := 0; i < len(v.verifierList); i++ {
		log.Info(modulName, "verifier Node", v.verifierList[i].ID)
		if v.isVerifierLeaderByNodeId(v.verifierList[i]) {
			continue
		}
		log.Info(modulName, "Tx verifier Node Success", v.verifierList[i].ID, "IP", v.verifierList[i].IP, "val", v.leaderRemoveTxList)
		go sendMsg(verifierToVerifier, msgSendVoteReqToFollower, v.leaderRemoveTxList, v.verifierList[i].IP, false)

	}
	return
}

func (v *Verifier) sendTxToLeader(txs []types.Transaction) {
	sendMsg(verifierToVerifier, msgSendTxToLeader, txs, v.leadNodeInfo.IP, true)
	return
}

func (v *Verifier) sendInvalidTxToLeader(invalidTx []uint16) {
	sendMsg(verifierToVerifier, msgSendInvalidTxToLeader, invalidTx, v.leadNodeInfo.IP, false)
	return
}

func (v *Verifier) sendVoteResultToLeader(vrList VoteResult) {
	sendMsg(verifierToVerifier, msgSendVoteResultToLeader, vrList, v.leadNodeInfo.IP, false)
	log.Info(modulName, "vr Info", vrList)
	return
}

func (v *Verifier) voteToTx(leaderInvalidTxList []uint16) {

	var srchCnt = 0

	log.Info(modulName, "followerInvalidList", v.followerInvalidTxsList, "leaderInvalideList", leaderInvalidTxList)
	for i := 0; i < len(v.followerInvalidTxsList); i++ {
		for j := 0; j < len(leaderInvalidTxList); j++ {
			if v.followerInvalidTxsList[i] == leaderInvalidTxList[j] {
				srchCnt++
				log.Info(modulName, "follower verify fail")
				break
			}
		}
	}

	var vr VoteResult
	if srchCnt != len(v.followerInvalidTxsList) {
		log.Info(modulName, "follower msg verifier fail\n", "", "invalid txs nums ", len(v.followerInvalidTxsList), "fail num")
		vr.Result = false
	} else {
		vr.Result = true
	}
	vr.NodeId = v.localNodeInfo.ID
	log.Info(modulName, "Vote to Transaction result", vr)
	v.sendVoteResultToLeader(vr)
}

const (
	follower uint8 = iota
	leader
)

func readySendNodelist(v *Verifier, blockNum uint64) {
	if nodeList, err := v.GenerateMainNodeList(blockNum); err == nil {
		data, _ := json.MarshalIndent(nodeList, "", "   ")
		v.sendMsgToMiner(data, Broadcast, blockNum)
		log.Info(modulName, "Leader To Miner Broadcast", blockNum)
	}
	return
}

type verifierList []election.NodeInfo

func (v verifierList) Len() int {
	return len(v)
}

func (v verifierList) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}

func (v verifierList) Less(i, j int) bool {
	if v[i].Wealth < v[j].Wealth {

		return v[i].Wealth < v[j].Wealth
	} else if v[i].Wealth == v[j].Wealth {
		id1, _ := strconv.Atoi(v[i].ID)
		id2, _ := strconv.Atoi(v[j].ID)
		return id1 < id2
	} else {
		return v[i].Wealth < v[j].Wealth
	}
}

func (verifier *Verifier) electionLeader(nodeList []election.NodeInfo, blockHeight uint64) {

	var sortNodeList = verifierList(nodeList)
	sort.Sort(sortNodeList)
	verifierNumber := uint64(len(nodeList))

	verifier.leadNodeInfo = sortNodeList[blockHeight%verifierNumber]

	if verifier.isVerifierLeaderByNodeId(verifier.localNodeInfo) {
		verifier.role = leader
		log.Info(modulName, "Become Leader", "")
	} else {
		verifier.role = follower
		log.Info(modulName, "Become Follower", "")
	}

}

func searchNodeInfoByNodeId(nodeList []election.NodeInfo, srchNodeId string) *election.NodeInfo {

	for i := 0; i < len(nodeList); i++ {
		if nodeList[i].ID == srchNodeId {
			return &nodeList[i]
		}
	}
	return nil
}

func (v *Verifier) ConfigNodelist(minerList []election.NodeInfo, verifierList []election.NodeInfo) {
	log.Info(modulName, "Config node list!", "")
	v.minerList = minerList
	v.verifierList = verifierList
	return
}
func (v *Verifier) Start(localNodeInfo *discover.Node) {
	log.Info(modulName, "Start verifier node!", "")
	if rslt := searchNodeInfoByNodeId(v.verifierList, localNodeInfo.ID.String()); rslt != nil {
		v.localNodeInfo = *rslt
		v.nodeState = nodeBusy
	} else {
		log.Info(modulName, "local node", *localNodeInfo, "isnt verifier", v.verifierList)
		v.nodeState = nodeIdle
	}
	return
}

func (v *Verifier) Stop() {
	v.nodeState = nodeIdle
	v.sessionPM.reset()
}

func (v *Verifier) Notify(blockHeigh uint64) {
	v.notifyChan <- blockHeigh
}
