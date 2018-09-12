package election

import (
	"fmt"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/election/ManElec100/mt19937"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/mc"
	//"github.com/ethereum/go-ethereum/mc"
)

/*
const (
	maxMinerNum              = params.MaxMinerNum              //支持的最大参选矿工数
	maxValidatorNum          = params.MaxValidatorNum          //支持的最大参选验证者数
	maxMasterMinerNum        = params.MaxMasterMinerNum        //顶层节点最大矿工主节点数
	maxMasterValidatorNum    = params.MaxMasterValidatorNum    //顶层节点最大矿工主节点数
	maxBackUpMinerNum        = params.MaxBackUpMinerNum        //最大备份矿工数
	maxCandidateMinerNum     = params.MaxCandidateMinerNum     //最大候补矿工数
	maxBackUpValidatorNum    = params.MaxBackUpValidatorNum    //最大备份验证者数
	maxCandidateValidatorNum = params.MaxCandidateValidatorNum //最大候补验证者数
)
*/

type ElectEventType string

type Mynode struct {
	Nodeid string
	Tps    int
	Stk    float64
	Uptime int
}

type foundnode struct {
	nodeid string
	tps    int
	stk    float64
	uptime int
}

type Elector struct {
	EleMMSub  ElectMMSub
	EleMVSub  ElectMVSub
	EleMMRs   chan mc.MasterMinerReElectionRsp
	EleMVRs   chan mc.MasterValidatorReElectionRsq
	Engine    func(probVal []stf, seed int64) ([]strallyint, []strallyint, []strallyint) //func(probVal map[string]float32, seed int64) ([]strallyint, []strallyint, []strallyint)
	msgcenter *mc.Center
	MaxSample int //配置参数,采样最多发生1000次,是一个离P+M较远的值
	J         int //基金会验证节点个数tps_weight
	M         int //验证主节点个数
	P         int //备份主节点个数
	N         int //矿工主节点个数
}

type ElectMMSub struct {
	MasterMinerReElectionReqMsgCH  chan mc.MasterMinerReElectionReqMsg
	MasterMinerReElectionReqMsgSub event.Subscription
}
type ElectMVSub struct {
	MasterValidatorReElectionReqMsgCH  chan mc.MasterValidatorReElectionReqMsg
	MasterValidatorReElectionReqMsgSub event.Subscription
}

func NewEle() *Elector {
	var ele Elector

	ele.MaxSample = 1000
	ele.J = 0
	ele.M = 11
	ele.P = 5
	ele.N = 21
	ele.EleServer()
	ele.EleMMRs = make(chan mc.MasterMinerReElectionRsp, 10)
	ele.EleMVRs = make(chan mc.MasterValidatorReElectionRsq, 10)

	return &ele
}

/////////////////////////////

type Self struct {
	nodeid   string
	stk      float64
	uptime   int
	tps      int
	Coef_tps float64
	Coef_stk float64
}

func (self *Self) TPS_POWER() float64 {
	tps_weight := 1.0
	if self.tps >= 16000 {
		tps_weight = 5.0
	} else if self.tps >= 8000 {
		tps_weight = 4.0
	} else if self.tps >= 4000 {
		tps_weight = 3.0
	} else if self.tps >= 2000 {
		tps_weight = 2.0
	} else if self.tps >= 1000 {
		tps_weight = 1.0
	} else {
		tps_weight = 0.0
	}
	return tps_weight
}

func (self *Self) Last_Time() float64 {
	CandidateTime_weight := 4.0
	if self.uptime <= 64 {
		CandidateTime_weight = 0.25
	} else if self.uptime <= 128 {
		CandidateTime_weight = 0.5
	} else if self.uptime <= 256 {
		CandidateTime_weight = 1
	} else if self.uptime <= 512 {
		CandidateTime_weight = 2
	} else {
		CandidateTime_weight = 4
	}
	return CandidateTime_weight
}

func (self *Self) deposit_stake() float64 {
	stake_weight := 1.0
	if self.stk >= 40000 {
		stake_weight = 4.5
	} else if self.stk >= 20000 {
		stake_weight = 2.15
	} else if self.stk >= 10000 {
		stake_weight = 1.0
	} else {
		stake_weight = 0.0
	}
	return stake_weight
}

type stf struct {
	Str  string
	Flot float32
}

func CalcAllValueFunction(nodelist []vm.DepositDetail) []stf { //nodelist []Mynode) map[string]float32 {
	//	CapitalMap := make(map[string]float64)
	//	CapitalMap := make(map[string]float32)
	var CapitalMap []stf
	var stk float64

	for _, item := range nodelist {
		stk = float64(item.Deposit.Uint64() / 1000000)
		self := Self{nodeid: string(item.NodeID[:]), stk: stk, uptime: int(item.OnlineTime.Uint64()), tps: 1000, Coef_tps: 0.2, Coef_stk: 0.25}
		value := self.Last_Time() * (self.TPS_POWER()*self.Coef_tps + self.deposit_stake()*self.Coef_stk)
		//		CapitalMap[self.nodeid] = float32(value)
		CapitalMap = append(CapitalMap, stf{Str: self.nodeid, Flot: float32(value)})
	}
	return CapitalMap
}

type pnormalized struct {
	Value  float32
	Nodeid string
}

type strallyint struct {
	Value  int
	Nodeid string
}

func Normalize(probVal []stf) []pnormalized {

	fmt.Println(probVal)
	var total float32
	//	var mlen int
	for _, item := range probVal {
		total += item.Flot
		//		fmt.Println("There are", views, "views for", key)
	}
	var pnormalizedlist []pnormalized
	for _, item := range probVal {
		var tmp pnormalized
		tmp.Value = item.Flot / total
		tmp.Nodeid = item.Str
		pnormalizedlist = append(pnormalizedlist, tmp)
		//		fmt.Println("There are", views, "views for", key)
	}
	return pnormalizedlist
}

func Sample1NodesInValNodes(probnormalized []pnormalized, rand01 float32) string {

	for _, iterm := range probnormalized {
		rand01 -= iterm.Value
		if float32(rand01) < iterm.Value {
			return iterm.Nodeid
		}
	}
	return probnormalized[0].Nodeid
}

func WriteWithFileWrite(name, content string) {
	fileObj, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		fmt.Println("Failed to open the file", err.Error())
		os.Exit(2)
	}
	defer fileObj.Close()
	if _, err := fileObj.WriteString(content); err == nil {
		fmt.Println("Successful writing to the file with os.OpenFile and *File.WriteString method.", content)
	}
}

func sortfunc(probnormalized []pnormalized, PricipalValNodes, BakValNodes, RemainingProbNormalizedNodes []strallyint) ([]strallyint, []strallyint, []strallyint) {
	Pricipal := make(map[string]int)
	BakVal := make(map[string]int)
	Remain := make(map[string]int)

	var RPricipalValNodes []strallyint
	var RBakValNodes []strallyint
	var RRemainingProbNormalizedNodes []strallyint

	for _, item := range PricipalValNodes {
		Pricipal[item.Nodeid] = item.Value
	}
	for _, item := range BakValNodes {
		BakVal[item.Nodeid] = item.Value
	}
	for _, item := range RemainingProbNormalizedNodes {
		Remain[item.Nodeid] = item.Value
	}

	for _, item := range probnormalized {
		var ok bool
		_, ok = Pricipal[item.Nodeid]
		if ok == true {
			RPricipalValNodes = append(RPricipalValNodes, strallyint{Nodeid: item.Nodeid, Value: Pricipal[item.Nodeid]})
			continue
		}

		_, ok = BakVal[item.Nodeid]
		if ok == true {
			RBakValNodes = append(RBakValNodes, strallyint{Nodeid: item.Nodeid, Value: BakVal[item.Nodeid]})
		}

		_, ok = Remain[item.Nodeid]
		if ok == true {
			RRemainingProbNormalizedNodes = append(RRemainingProbNormalizedNodes, strallyint{Nodeid: item.Nodeid, Value: Remain[item.Nodeid]})
		}

	}
	return RPricipalValNodes, RBakValNodes, RRemainingProbNormalizedNodes
}

func (Ele *Elector) SampleMPlusPNodes(probnormalized []pnormalized, seed int64) ([]strallyint, []strallyint, []strallyint) {
	var PricipalValNodes []strallyint
	var RemainingProbNormalizedNodes []strallyint //[]pnormalized
	var FirstMMinusJNodes []strallyint
	var BakValNodes []strallyint

	// 如果当选节点不到M-J个(加上基金会节点不足M个),则全部当选,其他列表为空
	//	var dict map[string]int
	dict := make(map[string]int)
	if len(probnormalized) <= Ele.M-Ele.J { //加判断 定义为func
		for _, item := range probnormalized {
			//			probnormalized[index].value = 100 * iterm.value
			temp := strallyint{Value: int(100 * item.Value), Nodeid: item.Nodeid}
			PricipalValNodes = append(PricipalValNodes, temp)
		}
		//		return [(e[0],int(100*e[1])) for e in probnormalized],[],[]
		return sortfunc(probnormalized, PricipalValNodes, BakValNodes, RemainingProbNormalizedNodes)
	}

	// 如果当选节点超过M-J,最多连续进行1000次采样或者选出M+P-J个节点
	rand := mt19937.RandUniformInit(seed)

	for i := 0; i < Ele.MaxSample; i++ {
		node := Sample1NodesInValNodes(probnormalized, float32(rand.Uniform(0.0, 1.0)))

		//		WriteWithFileWrite("D:/testdata/Sample1NodesInValNodes.txt", hex.EncodeToString([]byte(node)))
		//		WriteWithFileWrite("D:/testdata/Sample1NodesInValNodes.txt", "\n")
		_, ok := dict[node]

		if ok == true {
			dict[node] = dict[node] + 1
		} else {
			dict[node] = 1
		}

		if len(dict) == (Ele.M + Ele.P - Ele.J) {
			break
		} else if len(dict) == Ele.M-Ele.J && len(FirstMMinusJNodes) == 0 {
			for k, v := range dict {
				temp := strallyint{Value: v, Nodeid: k}
				FirstMMinusJNodes = append(FirstMMinusJNodes, temp) //todo: 直接转成map
				//				WriteWithFileWrite("D:/testdata/FirstMMinusJNodes.txt", hex.EncodeToString([]byte(temp.Nodeid)))
				//FirstMMinusJNodes := list(dict.keys())
			}
		}
	}
	// 如果没有选够M-J个或者刚好选够
	if len(dict) <= Ele.M-Ele.J {

		for _, item := range probnormalized {
			vint, ok := dict[item.Nodeid]

			if ok == true {
				var tmp strallyint
				tmp.Nodeid = item.Nodeid
				tmp.Value = vint
				PricipalValNodes = append(PricipalValNodes, tmp)
			} else {
				RemainingProbNormalizedNodes = append(RemainingProbNormalizedNodes, strallyint{Value: int(item.Value), Nodeid: item.Nodeid})
			}
		}
		//		return PricipalValNodes, BakValNodes, RemainingProbNormalizedNodes
		return sortfunc(probnormalized, PricipalValNodes, BakValNodes, RemainingProbNormalizedNodes)
	}

	// 如果选择超过M-J个
	tmpmap := make(map[string]int)
	for _, item := range FirstMMinusJNodes {
		tmpmap[item.Nodeid] = 0
	}

	for k, v := range dict {
		_, ok := tmpmap[k]
		if ok == true {
			var tmp strallyint
			tmp.Nodeid = k
			tmp.Value = v
			PricipalValNodes = append(PricipalValNodes, tmp)
		} else {
			BakValNodes = append(BakValNodes, strallyint{Value: v, Nodeid: k})
		}
	}

	for _, item := range probnormalized {
		_, ok := dict[item.Nodeid]
		if ok == false {
			RemainingProbNormalizedNodes = append(RemainingProbNormalizedNodes, strallyint{Nodeid: item.Nodeid, Value: int(item.Value)})
		}
	}
	//	return PricipalValNodes, BakValNodes, RemainingProbNormalizedNodes
	return sortfunc(probnormalized, PricipalValNodes, BakValNodes, RemainingProbNormalizedNodes)
}

func (Ele *Elector) SampleMinerNodes(probnormalized []pnormalized, seed int64, Ms int) ([]strallyint, []strallyint) {

	var PricipalMinerNodes []strallyint
	//	var RemainingProbNormalizedNodes []strallyint //[]pnormalized
	var BakMinerNodes []strallyint

	sort := func(probnormalized []pnormalized, PricipalMinerNodes []strallyint, BakMinerNodes []strallyint) ([]strallyint, []strallyint) {
		Pricipal := make(map[string]int)
		BakMin := make(map[string]int)

		var RPricipalMinerNodes []strallyint
		var RBakMinerNodes []strallyint

		for _, item := range PricipalMinerNodes {
			Pricipal[item.Nodeid] = item.Value
		}
		for _, item := range BakMinerNodes {
			BakMin[item.Nodeid] = item.Value
		}
		for _, item := range probnormalized {
			var ok bool
			_, ok = Pricipal[item.Nodeid]
			if ok == true {
				RPricipalMinerNodes = append(RPricipalMinerNodes, strallyint{Nodeid: item.Nodeid, Value: Pricipal[item.Nodeid]})
				continue
			}
			_, ok = BakMin[item.Nodeid]
			if ok == true {
				RBakMinerNodes = append(RBakMinerNodes, strallyint{Nodeid: item.Nodeid, Value: BakMin[item.Nodeid]})
			}
		}
		return RPricipalMinerNodes, RBakMinerNodes
	}

	// 如果当选节点不到N个,其他列表为空
	dict := make(map[string]int)
	Ele.N = Ms
	if len(probnormalized) <= Ele.N { //加判断 定义为func
		for _, item := range probnormalized {
			//			probnormalized[index].value = 100 * iterm.value
			temp := strallyint{Value: int(100 * item.Value), Nodeid: item.Nodeid}
			PricipalMinerNodes = append(PricipalMinerNodes, temp)
		}
		//		return [(e[0],int(100*e[1])) for e in probnormalized],[],[]
		return sort(probnormalized, PricipalMinerNodes, BakMinerNodes)
	}

	// 如果当选节点超过N,最多连续进行1000次采样或者选出N个节点
	rand := mt19937.RandUniformInit(seed)
	for i := 0; i < Ele.MaxSample; i++ {
		node := Sample1NodesInValNodes(probnormalized, float32(rand.Uniform(0.0, 1.0)))
		_, ok := dict[node]
		if ok == true {
			dict[node] = dict[node] + 1
		} else {
			dict[node] = 1
		}
		if len(dict) == Ele.N {
			break
		}
	}

	// 如果没有选够N个
	for _, item := range probnormalized {
		vint, ok := dict[item.Nodeid]

		if ok == true {
			var tmp strallyint
			tmp.Nodeid = item.Nodeid
			tmp.Value = vint
			PricipalMinerNodes = append(PricipalMinerNodes, tmp)
		} else {
			BakMinerNodes = append(BakMinerNodes, strallyint{Value: int(item.Value), Nodeid: item.Nodeid})
		}
	}
	lenPM := len(PricipalMinerNodes)
	if Ele.N > lenPM {
		PricipalMinerNodes = append(PricipalMinerNodes, BakMinerNodes[:Ele.N-lenPM]...)
		BakMinerNodes = BakMinerNodes[Ele.N-lenPM:]
	}
	return sort(probnormalized, PricipalMinerNodes, BakMinerNodes)
}

func CalcRemainingNodesVotes(RemainingProbNormalizedNodes []strallyint) []strallyint {
	for index, _ := range RemainingProbNormalizedNodes {
		RemainingProbNormalizedNodes[index].Value = 1
	}
	return RemainingProbNormalizedNodes
}

//做异常判断
func (Ele *Elector) ValNodesSelected(probVal []stf, seed int64) ([]strallyint, []strallyint, []strallyint) {

	probnormalized := Normalize(probVal)
	fmt.Println(probnormalized)
	PricipalValNodes, BakValNodes, RemainingProbNormalizedNodes := Ele.SampleMPlusPNodes(probnormalized, seed)

	// 计算所有剩余节点的股权
	RemainingValNodes := CalcRemainingNodesVotes(RemainingProbNormalizedNodes)

	//基金会节点加入验证主节点列表
	//如果验证主节点不足M个,使用剩余节点列表补足M-J个
	if len(PricipalValNodes) < Ele.M-Ele.J && len(RemainingValNodes) > 0 {
		for i := 0; Ele.M-Ele.J < i; i++ {
			PricipalValNodes = append(PricipalValNodes, RemainingValNodes[0])
		}
		PricipalValNodes = PricipalValNodes[1:]
	}

	// 如果备份主节点不足P个,使用剩余节点列表补足P个
	if len(BakValNodes) < Ele.P && len(RemainingValNodes) > 0 {
		for i := 0; Ele.M-Ele.J < i; i++ {
			BakValNodes = append(BakValNodes, RemainingValNodes[0])
		}
		PricipalValNodes = PricipalValNodes[1:]
	}
	return PricipalValNodes, BakValNodes, RemainingValNodes
}

func (Ele *Elector) MinerNodesSelected(probVal []stf, seed int64, Ms int) ([]strallyint, []strallyint) {
	probnormalized := Normalize(probVal)

	fmt.Println(probnormalized)
	PricipalMinerNodes, BakMinerNodes := Ele.SampleMinerNodes(probnormalized, seed, Ms)

	//计算所有剩余节点的股权
	BakMinerNodes = CalcRemainingNodesVotes(BakMinerNodes)
	return PricipalMinerNodes, BakMinerNodes
}

func (Ele *Elector) ChoiceEngine(flag int) {
	if flag == 1 {
		Ele.Engine = Ele.ValNodesSelected
	}
}

func (Ele *Elector) EleServer() {

	log.Info("Elector EleServer")
	//订阅消息
	Ele.EleMMSub = ElectMMSub{MasterMinerReElectionReqMsgCH: make(chan mc.MasterMinerReElectionReqMsg, 10)}
	Ele.EleMMSub.MasterMinerReElectionReqMsgSub, _ = mc.SubscribeEvent(mc.ReElec_MasterMinerReElectionReq, Ele.EleMMSub.MasterMinerReElectionReqMsgCH)

	Ele.EleMVSub = ElectMVSub{MasterValidatorReElectionReqMsgCH: make(chan mc.MasterValidatorReElectionReqMsg, 10)}
	Ele.EleMVSub.MasterValidatorReElectionReqMsgSub, _ = mc.SubscribeEvent(mc.ReElec_MasterValidatorElectionReq, Ele.EleMVSub.MasterValidatorReElectionReqMsgCH)

	//选择引擎
	Ele.ChoiceEngine(1)

	//开启监听
	go Ele.Listen()

	//返回消息
	go Ele.Post()
}

func (Ele *Elector) Post() {
	log.Info("Elector Post")
	for {
		select {
		case mmrers := <-Ele.EleMMRs:
			//			time.Sleep(5 * time.Second)
			log.Info("Elector Post", "Topo_MasterMinerElectionRsp", mmrers)
			err := mc.PublishEvent(mc.Topo_MasterMinerElectionRsp, mmrers) //mc.MasterMinerReElectionRspMsg{SeqNum: 666})
			log.Info("Post发送状态", err)

		case mvrers := <-Ele.EleMVRs:
			log.Info("Elector Post", "Topo_MasterValidatorElectionRsp", mvrers)
			err1 := mc.PublishEvent(mc.Topo_MasterValidatorElectionRsp, mvrers) //mc.MasterValidatorReElectionRspMsg{SeqNum: 666})
			log.Info("Post发送状态", err1)
		}
	}
}

func (Ele *Elector) Listen() {

	defer Ele.EleMMSub.MasterMinerReElectionReqMsgSub.Unsubscribe()
	defer Ele.EleMVSub.MasterValidatorReElectionReqMsgSub.Unsubscribe()

	log.Info("Elector Listen")
	for {
		select {
		case mmrerm := <-Ele.EleMMSub.MasterMinerReElectionReqMsgCH:
			MinerElectMap := make(map[string]vm.DepositDetail)
			for i, item := range mmrerm.MinerList {
				//				MinerElectMap[string(item.Account[:])] = item
				MinerElectMap[string(item.NodeID[:])] = item
				if item.Deposit == nil {
					mmrerm.MinerList[i].Deposit = big.NewInt(50000)
				}
				if item.WithdrawH == nil {
					mmrerm.MinerList[i].WithdrawH = big.NewInt(0)
				}
				if item.OnlineTime == nil {
					mmrerm.MinerList[i].OnlineTime = big.NewInt(300)
				}
			}

			value := CalcAllValueFunction(mmrerm.MinerList)

			a, b := Ele.MinerNodesSelected(value, mmrerm.RandSeed.Int64(), 21) //Ele.Engine(value, mmrerm.RandSeed.Int64()) //0x12217)
			for index, item := range a {
				fmt.Println(index, item)
			}
			for index, item := range b {
				fmt.Println(index, item)
			}
			var MinerEleRs mc.MasterMinerReElectionRsp
			MinerEleRs.SeqNum = mmrerm.SeqNum

			for index, item := range a {
				fmt.Println(item.Nodeid, []byte(item.Nodeid))
				tmp := MinerElectMap[item.Nodeid]
				var ToG mc.TopologyNodeInfo
				ToG.Account = tmp.Address
				ToG.Position = uint16(index)
				ToG.Type = common.RoleMiner
				ToG.Stock = uint16(item.Value)
				MinerEleRs.MasterMiner = append(MinerEleRs.MasterMiner, ToG)
			}

			for index, item := range b {
				tmp := MinerElectMap[item.Nodeid]
				var ToG mc.TopologyNodeInfo
				ToG.Account = tmp.Address
				//				ToG.OnlineState = true
				ToG.Position = uint16(index)
				ToG.Type = common.RoleMiner
				ToG.Stock = uint16(item.Value)
				MinerEleRs.BackUpMiner = append(MinerEleRs.BackUpMiner, ToG)
			}

			Ele.EleMMRs <- MinerEleRs
			//			Ele.EleMVRs <- fmt.Println(value)
			//fmt.Println("收到数据", mmrerm)

		case mvrerm := <-Ele.EleMVSub.MasterValidatorReElectionReqMsgCH:
			log.Info("Elector Listen", "ReElec_MasterValidatorElectionReq", mvrerm)
			ValidatorElectMap := make(map[string]vm.DepositDetail)
			for i, item := range mvrerm.ValidatorList {
				ValidatorElectMap[string(item.NodeID[:])] = item
				//todo: panic
				if item.Deposit == nil {
					mvrerm.ValidatorList[i].Deposit = big.NewInt(50000)
				}
				if item.WithdrawH == nil {
					mvrerm.ValidatorList[i].WithdrawH = big.NewInt(0)
				}
				if item.OnlineTime == nil {
					mvrerm.ValidatorList[i].OnlineTime = big.NewInt(300)
				}
			}

			value := CalcAllValueFunction(mvrerm.ValidatorList)
			a, b, c := Ele.Engine(value, mvrerm.RandSeed.Int64()) //0x12217)
			var ValidatorEleRs mc.MasterValidatorReElectionRsq
			ValidatorEleRs.SeqNum = mvrerm.SeqNum
			for index, item := range a {
				tmp := ValidatorElectMap[item.Nodeid]
				var ToG mc.TopologyNodeInfo
				ToG.Account = tmp.Address
				ToG.Position = uint16(index)
				ToG.Type = common.RoleValidator
				ToG.Stock = uint16(item.Value)
				ValidatorEleRs.MasterValidator = append(ValidatorEleRs.MasterValidator, ToG)
			}

			for index, item := range b {
				tmp := ValidatorElectMap[item.Nodeid]
				var ToG mc.TopologyNodeInfo
				ToG.Account = tmp.Address
				ToG.Position = uint16(index)
				ToG.Type = common.RoleValidator
				ToG.Stock = uint16(item.Value)
				ValidatorEleRs.BackUpValidator = append(ValidatorEleRs.BackUpValidator, ToG)
			}

			for index, item := range c {
				tmp := ValidatorElectMap[item.Nodeid]
				var ToG mc.TopologyNodeInfo
				ToG.Account = tmp.Address

				ToG.Position = uint16(index)
				ToG.Type = common.RoleValidator
				ToG.Stock = uint16(item.Value)
				ValidatorEleRs.CandidateValidator = append(ValidatorEleRs.CandidateValidator, ToG)
			}
			Ele.EleMVRs <- ValidatorEleRs
			//			Ele.EleMVRs <- fmt.Println(value)
			//	fmt.Println("受到数据", mvrerm)
		}
	}
}

func (Ele *Elector) ToPoUpdate(Q0, Q1, Q2 []mc.TopologyNodeInfo, nettopo mc.TopologyGraph, offline []common.Address) []mc.Alternative {

	log.Info("Elector ToPoUpdate")
	netmap := make(map[common.Address]mc.TopologyNodeInfo)
	Q0map := make(map[common.Address]mc.TopologyNodeInfo)
	Q1map := make(map[common.Address]mc.TopologyNodeInfo)
	Q2map := make(map[common.Address]mc.TopologyNodeInfo)

	for _, item := range Q0 {
		Q0map[item.Account] = item
	}

	for _, item := range Q1 {
		Q1map[item.Account] = item
	}

	for _, item := range Q2 {
		Q2map[item.Account] = item
	}

	for _, item := range nettopo.NodeList {
		netmap[item.Account] = item
	}

	for _, item := range nettopo.NodeList {
		var ok bool
		_, ok = Q0map[item.Account]
		if ok == true {
			delete(Q0map, item.Account)
		}

		_, ok = Q1map[item.Account]
		if ok == true {
			delete(Q1map, item.Account)
		}

		_, ok = Q2map[item.Account]
		if ok == true {
			delete(Q2map, item.Account)
		}
	}

	var substitute []mc.TopologyNodeInfo

	for _, v := range Q0map {
		substitute = append(substitute, v)
	}
	for _, v := range Q1map {
		substitute = append(substitute, v)
	}
	for _, v := range Q2map {
		substitute = append(substitute, v)
	}
	var sublen = len(substitute)

	var alternalist []mc.Alternative

	for index, item := range offline {
		if index < sublen {
			tmp := netmap[item]
			var talt mc.Alternative
			talt.B = item
			talt.A = substitute[index].Account
			talt.Position = tmp.Position

			alternalist = append(alternalist, talt)
		} else {
			break
		}
	}
	return alternalist
}

func (Ele *Elector) PrimarylistUpdate(Q0, Q1, Q2 []mc.TopologyNodeInfo, online mc.TopologyNodeInfo, flag int) ([]mc.TopologyNodeInfo, []mc.TopologyNodeInfo, []mc.TopologyNodeInfo) {
	log.Info("Elector PrimarylistUpdate")
	if flag == 0 {
		var tQ0 []mc.TopologyNodeInfo
		tQ0 = append(tQ0, online)
		tQ0 = append(tQ0, Q0...)
		Q0 = tQ0
	}

	if flag == 1 {
		var tQ1 []mc.TopologyNodeInfo
		tQ1 = append(tQ1, Q1...)
		tQ1 = append(tQ1, online)
		Q1 = tQ1
	}

	if flag == 2 {
		var tQ2 []mc.TopologyNodeInfo
		tQ2 = append(tQ2, Q2...)
		tQ2 = append(tQ2, online)
		Q2 = tQ2
	}
	return Q0, Q1, Q2
}
