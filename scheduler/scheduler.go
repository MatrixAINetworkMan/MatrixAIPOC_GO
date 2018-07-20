package scheduler

import (
	"context"
	"math/big"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/election"
	"github.com/ethereum/go-ethereum/election/manhash"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/miner"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/verify"
)

const (
	electionNetEffterTime = 6
	blockEffectDelay      = 4
)

const (
	SUPERMINER  = iota
	MINER
	SUPERVERIFY
	VERIFY
	ORDINATY
)

type INode interface {
	start()
}

type Ipinfo struct {
	Ip           string
	Protocoltype uint8 //0:UDP,1:TCP,2:VNP
}

const NODEMAXNUM = 20000

type Scheduler struct {
	running        bool
	nodetype       int        //0:矿工节点，1：一般矿工，2：验证节点，3：一般验证节点,4:钱包节点，-1非法节点
	prenodetype    int        //0:矿工节点，1：一般矿工，2：验证节点，3：一般验证节点,4:钱包节点，-1非法节点
	ch2            chan int32 //区块同步后向网络拓扑模块写入主节点列表
	BlockInsertch  chan bool  //插入区块通知
	nodeList       []*election.NodeList
	lock           sync.RWMutex
	bc             *core.BlockChain
	miner          *miner.Miner
	ele            []*manhash.Election //选举对象
	eleEffterIndex int
	eletempIndex   int
	Node           *discover.Node
	ethClient      *ethclient.Client
	accountManager *accounts.Manager
	verifier       *verifier.Verifier
	chainConfig    *params.ChainConfig
	Startmining    func()
	Stoptmining    func()
}

func getrandomleafnode() (ip []Ipinfo, err error) {
	return ip, nil

}

func getsubnode() (ip []Ipinfo, err error) {
	return ip, nil
}

func (self *Scheduler) getnodelistfrombootnodes() (err error) {

	self.nodeList[self.eletempIndex].MinerList = make([]election.NodeInfo, 2)
	self.nodeList[self.eletempIndex].CommitteeList = make([]election.NodeInfo, 1)

	node, _ := discover.ParseNode(params.MainnetBootnodes[0])

	self.nodeList[self.eletempIndex].CommitteeList[0].ID = node.ID.String()
	self.nodeList[self.eletempIndex].CommitteeList[0].IP = node.IP.String()
	self.nodeList[self.eletempIndex].CommitteeList[0].Wealth = 10000
	for i := 0; i < 2; i++ {
		node, _ := discover.ParseNode(params.MainnetBootnodes[i+1])

		self.nodeList[self.eletempIndex].MinerList[i].ID = node.ID.String()
		self.nodeList[self.eletempIndex].MinerList[i].IP = node.IP.String()
		self.nodeList[self.eletempIndex].MinerList[i].Wealth = 10000
	}
	log.Info("getnodelistfrombootnodes", "list:", self.nodeList[self.eletempIndex])
	return nil

}

func (self *Scheduler) startconnect() {

	//var iplist []Ipinfo
	var ch2 chan []Ipinfo //向网络拓扑模块写入主节点列表
	ch2 = make(chan []Ipinfo)
	iplist := make([]Ipinfo, 3, NODEMAXNUM)
	switch self.nodetype {
	case 0:
	case 2:
		supernodeinfo := self.ele[self.eleEffterIndex].GetSuperMiner()

		//elf.ele.GetChildIP()
		//go vpnconnect(iplist)
		temp := self.ele[self.eleEffterIndex].GetSuperCommittee()
		supernodeinfo = append(temp)
		supernodeinfo = supernodeinfo
		//go vpnconnect(iplist)
		//temp , _= getsubnode()
		//ip:=self.ele.GetChildIP()
		//templist := make([]Ipinfo, 2)
		//for i,v:=range ip{
		//	templist[i].Ip = v
		//	templist[i].Protocoltype=1
		//}
		//iplist = append(templist)
		//go tcpconnect(iplist)
		//ch2 <-iplist
	case 1:
	case 3:
		iplist, _ = getsubnode()

		childnodeinfo := self.ele[self.eleEffterIndex].GetChild(self.Node.ID.String())
		childnodeinfo = childnodeinfo
		//go tcpconnect(iplist)
		//ch2 <-iplist
	case 4:
		iplist, _ = getrandomleafnode()
		//go tcpconnect(iplist)
		//生成网络拓扑普通节点流程
		//iplist, _ := get2randomjudgesupernode()
		leafnodeinfo := self.ele[self.eleEffterIndex].GetLeafNode(self.Node.ID.String())
		leafnodeinfo = leafnodeinfo
		//iplist = append(temp)
		ch2 <- iplist
	default:
		//断开所有连接
	}
}

func (self *Scheduler) startTask() {
	//if self.nodetype == self.prenodetype {
	//	return
	//}
	self.verifier.ConfigNodelist(self.ele[self.eleEffterIndex].GetSuperMiner(), self.ele[self.eleEffterIndex].GetSuperCommittee())
	//0:矿工节点，1：一般矿工，2：验证节点，3：一般验证节点,4:钱包节点 stop
	switch self.prenodetype {
	case SUPERMINER:
		self.Stoptmining()
	case MINER:
	case SUPERVERIFY:
		//verifier stop
		self.verifier.Stop()
	case VERIFY:
	case ORDINATY:

	default:
	}
	//0:矿工节点，1：一般矿工，2：验证节点，3：一般验证节点,4:钱包节点 stop+
	switch self.nodetype {
	case SUPERMINER:
		log.Info("superminer start")
		self.Startmining()
		//go self.miner.Start(self.accountManager.Wallets()[0].Accounts()[0].Address)
	case MINER:
	case SUPERVERIFY:
		log.Info("supercommittee start")
		self.verifier.Start(self.Node)
	case VERIFY:
	case ORDINATY:

	default:
	}
	self.prenodetype = self.nodetype
}

func calcbcnodeblocknumber(n uint64) uint64 {
	//初始化流程如果小于生效时间使用前一个广播区块，
	if n%params.BroadcastInterval < electionNetEffterTime {
		n = (n/params.BroadcastInterval - 1) * params.BroadcastInterval
	} else {
		n = n - n%params.BroadcastInterval
	}
	return n
}

func New(bc *core.BlockChain, miner *miner.Miner, chainConfig *params.ChainConfig, verifier *verifier.Verifier) *Scheduler {
	Scheduler := new(Scheduler)
	Scheduler.running = false
	Scheduler.bc = bc
	Scheduler.miner = miner
	Scheduler.verifier = verifier
	Scheduler.ele = make([]*manhash.Election, 2)
	Scheduler.nodeList = make([]*election.NodeList, 2)
	Scheduler.nodeList[0] = &election.NodeList{nil, nil, nil, nil}
	Scheduler.nodeList[1] = &election.NodeList{nil, nil, nil, nil}
	Scheduler.ele[0] = new(manhash.Election)
	Scheduler.ele[1] = new(manhash.Election)
	Scheduler.eleEffterIndex = 0
	Scheduler.eletempIndex = 1
	Scheduler.prenodetype = 4
	Scheduler.nodetype = -1 //非法节点
	Scheduler.ch2 = make(chan int32, 1)
	Scheduler.BlockInsertch = make(chan bool, 1)
	Scheduler.bc.InserBlockNotify2Schedeuler(Scheduler.Setblockinsernotify)
	Scheduler.chainConfig = chainConfig
	//log.Info("Scheduler New","bc:",bc,"id:",discover.PubkeyID(&srv.PrivateKey.PublicKey))

	return Scheduler
}

func (bc *Scheduler) copyblockNodeList(block *types.Block, to *election.NodeList) (e *election.NodeList) {
	if 0 != len(block.Header().MinerList) {
		to.MinerList = make([]election.NodeInfo, len(block.Header().MinerList))
		copy(to.MinerList, block.Header().MinerList)

	}
	if 0 != len(block.Header().CommitteeList) {
		to.CommitteeList = make([]election.NodeInfo, len(block.Header().CommitteeList))
		copy(to.CommitteeList, block.Header().CommitteeList)
	}
	if 0 != len(block.Header().Both) {
		to.Both = make([]election.NodeInfo, len(block.Header().Both))
		copy(to.Both, block.Header().Both)
	}

	if 0 != len(block.Header().OfflineList) {
		to.OfflineList = make([]election.NodeInfo, len(block.Header().OfflineList))
		copy(to.OfflineList, block.Header().OfflineList)
	}

	return to
}

func (self *Scheduler) offlineListmortgagedeal(offlinelist [] election.NodeInfo, BlockCache map[string][]*types.Transaction) (err error) {

	for _, v := range offlinelist {
		log.Info("offlinelist", "range:", v.Account.String())
		account := v.Account
		accountsString := v.Account.String()
		transcations, ok := BlockCache[accountsString]
		//选举出来的退选 账户在没有给抵押账户转过钱，不处理
		if ok {

			amounts := big.NewInt(0)
			_, electType := transcations[len(transcations)-1].GetElectType()
			if types.ElectExit != electType {
				log.Info("Scheduler", "is not ElectExit:", v.Account.String())
				continue
			}

			Amount := transcations[len(transcations)-1].Value()
			amounts = Amount.Add(amounts, Amount)
			for i := len(transcations) - 2; i > -1; i-- {
				_, electType := transcations[i].GetElectType()
				if types.ElectExit == electType {
					break
				}

				Amount := transcations[i].Value()
				amounts = Amount.Add(amounts, Amount)
			}

			if big.NewInt(0) != amounts {
				myaccount := self.GetHypothecatedAccount().Address
				log.Info("Scheduler", "myaccount", myaccount)
				nonce, _ := self.ethClient.PendingNonceAt(context.Background(), myaccount)
				log.Info("Scheduler:", "nonce", nonce)
				log.Info("Scheduler:", "amounts", amounts.Uint64())

				tx1 := types.NewTransaction(nonce, account, amounts, 21000, big.NewInt(0), nil)

				log.Info("Scheduler:", "tx1", tx1)
				ks := self.accountManager.Backends(keystore.KeyStoreType)[0]

				err := ks.(*keystore.KeyStore).Unlock(*self.GetHypothecatedAccount(), "xxx")
				log.Info("Unlocked account", "err", err)
				signed, err := ks.(*keystore.KeyStore).SignTx(*self.GetHypothecatedAccount(), tx1, self.chainConfig.ChainID)

				log.Info("Scheduler:", "signed error", err)

				err = self.ethClient.SendTransaction(context.Background(), signed)
				log.Info("Scheduler :", "SendTransaction", err)
			}

		} else {
			log.Info("accountsString no exit:", "list", v)
		}
	}

	return nil
}

func (self *Scheduler) Start(Node *discover.Node, ethClient *ethclient.Client, accountManager *accounts.Manager, Startmining func(), Stoptmining func()) {
	//var ch0 chan types.Block //主节点或特殊节点区块通知

	//var ch2 chan []nodeInfo
	//ch2 := make(chan []nodeInfo)//区块同步后向网络拓扑模块写入主节点列表
	ch1 := make(chan bool, 1) // 网络拓扑生成通知
	self.Node = Node
	self.ethClient = ethClient
	self.accountManager = accountManager
	self.Startmining = Startmining
	self.Stoptmining = Stoptmining
	//var block types.Block
	self.running = true
	log.Info("Scheduler Start:", "nodeid:", Node.ID.String())
	//将网络拓扑ping-pong buffer传入网络拓扑生成模块
	//inserNetworkMemroy()
	//self.nodelist = make([]nodeInfo,3,NODEMAXNUM)
	//go self.miner.Start(self.accountManager.Wallets()[0].Accounts()[0].Address)
	for {
		select {
		case <-self.BlockInsertch:
			//区块插入消息
			blockNum := self.bc.CurrentBlock().NumberU64()
			log.Info("Scheduler:", "NumberU64", blockNum)

			//根据区块高度值作为时间驱动，产生选举处理
			if blockNum > params.BroadcastInterval {
				if blockNum%params.BroadcastInterval == blockEffectDelay {

					block := self.bc.GetBlockByNumber(uint64(blockNum - blockEffectDelay))
					//获取主节点列表生成网络拓扑,其中广播区块是在分叉时序通知，主节点区块是在区块同步完成
					log.Info("EffectDelay  block", "blockNum:", blockNum-blockEffectDelay)
					log.Info("EffectDelay  block", "block:", block)
					self.nodeList[self.eletempIndex] = self.copyblockNodeList(block, self.nodeList[self.eletempIndex])

					//log.Info("account", self.accountManager.Wallets()[0].Accounts()[0].Address.String())
					log.Info("EffectDelay U", "eletempIndex:", self.eletempIndex)
					log.Info("EffectDelay  nodelist", "nodelist", self.nodeList[self.eletempIndex].Both)

					//网络拓扑模块需要copy主节点列表，防止生成过程中有更新
					//eleNodeList := genElenodelist(block.Nodelist)
					log.Info("Scheduler GenNetwork")
					go self.ele[self.eletempIndex].GenNetwork(self.nodeList[self.eletempIndex], ch1)
				} else if blockNum%params.BroadcastInterval == electionNetEffterTime {
					//block := self.bc.CurrentBlock()
					//获取主节点列表生成网络拓扑,其中广播区块是在分叉时序通知，主节点区块是在区块同步完成
					//templist, _ := getnodelistfromblock(block)
					//to do：主节点列表提取出来需要延迟更新生成网络拓扑图-'
					//ping pong交换
					self.lock.Lock()
					self.eletempIndex, self.eleEffterIndex = self.eleEffterIndex, self.eletempIndex
					self.lock.Unlock()

					block := self.bc.GetBlockByNumber(uint64(10))
					log.Info(" block10", "data:", block)
					log.Info("Scheduler Update GenNetwork", "eletempIndex:", self.eletempIndex)

					log.Info("eletempIndex", "nodelist", self.nodeList[self.eletempIndex])
					//self.updatemainnodelist(self.nodeList[self.eletempIndex])
					log.Info("Scheduler Update GenNetwork", "eleEffterIndex:", self.eleEffterIndex)

					log.Info("electionNetEffterTime", "nodelist", self.nodeList[self.eleEffterIndex])

					self.nodetype = self.ele[self.eleEffterIndex].GetIDType(self.Node.ID.String())
					log.Info("nodetype", "value:", self.nodetype)

					self.startconnect()
					self.startTask() //如果存在创建的对象传入角色的对象，通过对象启动对应的服务
					if 0 == len(self.accountManager.Wallets()) || 0 == len(self.accountManager.Wallets()[0].Accounts()) {
						log.Info("no account")
						continue
					}

					account := self.GetHypothecatedAccount()
					if account != nil && strings.EqualFold(account.Address.String(), params.HypothecatedAccount) {
						self.offlineListmortgagedeal(self.nodeList[self.eleEffterIndex].OfflineList, self.bc.HACache)
					}
				}
			}
			if SUPERVERIFY == self.nodetype {
				self.verifier.Notify(blockNum)
			}
			continue
		case <-self.ch2:
			//区块同步后向网络拓扑模块写入主节点列表
			blockNum := self.bc.CurrentBlock().NumberU64()

			//第一个广播周期生效前主节点列表是boot节点
			if blockNum < params.BroadcastInterval+electionNetEffterTime {
				log.Info("CurrentBlock.NumberU64:", "NumberU64", blockNum)
				//d
				self.getnodelistfrombootnodes()
				log.Info("templist", "list:", self.nodeList[self.eletempIndex])

			} else {
				//todo:在electionNetEffterTime和blockEffectDelay生成两个列表和网络拓扑图，暂时在生效时间用前一个列表和网络拓扑图，

				log.Info("scheduler", "CurrentBlock.NumberU64:", blockNum)
				//从广播区块或更新区块里获取主节点列表
				blockNum = calcbcnodeblocknumber(blockNum)
				block := self.bc.GetBlockByNumber(blockNum)

				self.copyblockNodeList(block, self.nodeList[self.eletempIndex])

				log.Info("copyblockNodeList", "list:", self.nodeList[self.eletempIndex])

			}

			self.lock.Lock()
			self.eletempIndex, self.eleEffterIndex = self.eleEffterIndex, self.eletempIndex
			self.lock.Unlock()
			log.Info("nodelsit", "list:", self.nodeList[self.eleEffterIndex])
			//网络拓扑模块需要copy主节点列表，防止生成过程中有更新
			go self.ele[self.eleEffterIndex].GenNetwork(self.nodeList[self.eleEffterIndex], ch1)
			<-ch1
			//生成网络拓流程
			self.nodetype = self.ele[self.eleEffterIndex].GetIDType(self.Node.ID.String())
			log.Info("nodetype", "value:", self.nodetype)

			self.startconnect()

			self.startTask()

		default:

		}

	}
}

// 获取抵押账户
func (self *Scheduler) GetHypothecatedAccount() *accounts.Account {
	wallets := self.accountManager.Wallets()
	if len(wallets) <= 0 {
		return nil
	}

	accountList := wallets[0].Accounts()
	if len(accountList) <= 0 {
		return nil
	}

	return &accountList[0]
}

func (self *Scheduler) Getmainnodelist() (Nodelsit []election.NodeInfo, err error) {
	//temp := new(election.NodeList)
	self.lock.RLock()
	defer self.lock.RUnlock()
	minerlist := self.nodeList[self.eleEffterIndex].MinerList
	committeeList := self.nodeList[self.eleEffterIndex].CommitteeList
	bothlist := self.nodeList[self.eleEffterIndex].Both

	if 0 == len(minerlist)+len(committeeList)+len(bothlist) {
		return nil, nil
	}
	temp := make([]election.NodeInfo, 0, len(minerlist)+len(committeeList)+len(bothlist))
	//self.copyNodeList(self.nodeList[self.eleEffterIndex],temp)
	temp = append(temp, minerlist...)
	temp = append(temp, committeeList...)
	temp = append(temp, bothlist...)
	return temp, nil

}

func (self *Scheduler) Setmainnodelistnotify() (err error) {
	log.Info("Setmainnodelistnotify")
	self.ch2 <- 1
	return nil

}

func (self *Scheduler) Setblockinsernotify() {
	if self.running {
		log.Info("Setblockinsernotify")
		self.BlockInsertch <- true
	}
}
