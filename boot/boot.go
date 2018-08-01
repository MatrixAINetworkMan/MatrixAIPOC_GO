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
package boot

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum/go-ethereum/election"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/scheduler"
)

var (
	Public_Seq  uint64
	Need_Ack    []uint64
	go_bootss   bootss
	boot_FUN_RE RE_boot
	LocalIP     string
)

type Boot_Node_info struct {
	Nodeid string
	Ip     string
}

const (
	Time_Out_Limit            = 10 * time.Second
	Find_Conn_Status_Interval = 5
)

type GetHeightReq struct { //0x0001

}
type GetHeightRsp struct { //0x0002

	BlockHeight uint64
}
type GetMainNodeReq struct { //0x0003

}
type GetMainNodeRsp struct { //0x0004

	MainNode []election.NodeInfo
}

type RE_boot struct {
	Net_Flag  int
	Height    uint64
	Main_List []election.NodeInfo
}

type LocalHeightInfo struct {
	Ip  string
	Len uint64
}

type Bc_Scheduler struct {
	P_blockchain *core.BlockChain
	P_scheduler  *scheduler.Scheduler
}

type bootss struct {
	Mine_Bc_Scheduler Bc_Scheduler
	ChanSend          chan []p2p.Custsend
	ChanHeight        chan LocalHeightInfo
	ChanMainNode      chan []election.NodeInfo
}

func (this *bootss) Read_Chan() {
	for {
		res := <-go_bootss.ChanSend
		p2p.CustSend(res)
	}
}

func getip() (string, error) {
	conn, err := net.Dial("udp", "google.com:80")
	if err != nil {
		return "", fmt.Errorf("can not get loacl ip")
	}
	defer conn.Close()
	ans := strings.Split(conn.LocalAddr().String(), ":")[0]
	return ans, nil
}

func Is_in_Need_Ack(index uint64) int {
	for i := 0; i < len(Need_Ack); i++ {
		if Need_Ack[i] == index {
			Need_Ack = append(Need_Ack[:i], Need_Ack[i+1:]...)
			return i
		}
	}
	return -1
}

func (this *bootss) Get_local_block_height() uint64 {
	ss := go_bootss.Mine_Bc_Scheduler.P_blockchain.CurrentBlock().Header().Number.Uint64()
	return ss
}
func (this *bootss) Get_Mine_Main_Node() []election.NodeInfo {
	ss, _ := go_bootss.Mine_Bc_Scheduler.P_scheduler.Getmainnodelist()
	return ss
}

func Attept_Check_Not_Big_Than_Me(ListIp []string) {
	for {
		count := 0
		var ListIp_Conn_Status []p2p.Status_Re = p2p.CustStat(ListIp)
		for i := 0; i < len(ListIp_Conn_Status); i++ {
			if ListIp_Conn_Status[i].Ip_status {
				count++
			}
		}
		fmt.Println("ListIp_Conn_Status", ListIp_Conn_Status)
		if count == 2 {
			var ss_re []uint64
			var Status_Re int
			Status_Re, ss_re = go_bootss.Get_Block_Height(ListIp)
			if Status_Re == 1 {
				if ss_re[0] >= go_bootss.Get_local_block_height() || ss_re[1] >= go_bootss.Get_local_block_height() {
					var Main_List [][]election.NodeInfo
					Status_Re, Main_List = go_bootss.Get_Main_Node(ListIp)
					if Status_Re == 1 {
						boot_FUN_RE.Net_Flag = 1
						boot_FUN_RE.Height = ss_re[0]
						boot_FUN_RE.Main_List = Main_List[0]

						break
					}

				}
			}
		}
		time.Sleep(Find_Conn_Status_Interval * time.Second)
		log.Info("BOOT", "default node   sleep end  three node height not all zero")

	}

}

func Not_Default_Boot(ListIp []string) {
	for {
		count := 0
		var ListIp_Conn_Status []p2p.Status_Re = p2p.CustStat(ListIp)
		for i := 0; i < len(ListIp_Conn_Status); i++ {
			if ListIp_Conn_Status[i].Ip_status {
				count++
			}
		}
		fmt.Println("ListIp_Conn_Status", ListIp_Conn_Status)
		if count >= 2 {
			var ss_re []uint64
			var Status_Re int
			var Chose_Two_Boot []string
			for i := 0; i < len(ListIp_Conn_Status); i++ {
				if ListIp_Conn_Status[i].Ip_status {
					Chose_Two_Boot = append(Chose_Two_Boot, ListIp_Conn_Status[i].Ip)
				}
				if len(Chose_Two_Boot) >= 2 {
					break
				}

			}

			Status_Re, ss_re = go_bootss.Get_Block_Height(Chose_Two_Boot)
			if Status_Re == 1 {
				if ss_re[0] >= go_bootss.Get_local_block_height() || ss_re[1] >= go_bootss.Get_local_block_height() {
					var Main_List [][]election.NodeInfo
					Status_Re, Main_List = go_bootss.Get_Main_Node(Chose_Two_Boot)
					if Status_Re == 1 {
						boot_FUN_RE.Net_Flag = 1
						boot_FUN_RE.Height = ss_re[0]
						boot_FUN_RE.Main_List = Main_List[0]
						break
					}
				}
			}
		}
		time.Sleep(Find_Conn_Status_Interval * time.Second)
		log.Info("BOOT", "not default   sleep end   find two boot node")
	}
}

func Default_Boot_Find_Two_Boot(ListIp []string) int {
	for {
		count := 0
		var ListIp_Conn_Status []p2p.Status_Re = p2p.CustStat(ListIp)
		for i := 0; i < len(ListIp_Conn_Status); i++ {
			if ListIp_Conn_Status[i].Ip_status {
				count++
			}
		}
		fmt.Println("ListIp_Conn_Status", ListIp_Conn_Status)
		if count >= 2 {
			var Re_Height_List []uint64
			var Status_Re_Height int
			Status_Re_Height, Re_Height_List = go_bootss.Get_Block_Height(ListIp)
			if Status_Re_Height == 1 {
				if Re_Height_List[0] == 0 && Re_Height_List[1] == 0 && go_bootss.Get_local_block_height() == 0 {
					return 1
				} else {
					return 0
				}
			}

		}
		time.Sleep(Find_Conn_Status_Interval * time.Second)
		log.Info("BOOT", "default node   sleep end   heck other is zero")
	}
	return 0
}

func Make_Send_Msg(fromip string, toip string, istcp bool, code uint64, Type uint64) p2p.Custsend {
	var t p2p.Custsend
	t.FromIp = fromip
	t.ToIp = toip
	t.IsTcp = istcp
	t.Code = code

	var t_data_format p2p.Data_Format
	t_data_format.Type = Type
	t_data_format.Seq = Public_Seq
	Public_Seq = Public_Seq + 1

	switch Type {
	case 0x0001:
		var tt GetHeightReq
		t_data_format.Data_struct, _ = json.MarshalIndent(tt, "", "   ")
	case 0x0003:
		var tt GetMainNodeReq
		t_data_format.Data_struct, _ = json.MarshalIndent(tt, "", "   ")
	default:
		log.Info("BOOT", "Make_Send_Msg err Type=", Type)
	}
	t.Data = t_data_format
	return t

}
func (this *bootss) Get_Main_Node(ListIp_string []string) (int, [][]election.NodeInfo) { //获取主节点信息

	var t_Send []p2p.Custsend
	var Me_Need_Ack []uint64
	for i := 0; i < len(ListIp_string); i++ {
		t := Make_Send_Msg(LocalIP, ListIp_string[i], true, 1, 0x0003)
		t_Send = append(t_Send, t)
		Need_Ack = append(Need_Ack, t.Data.Seq)
		Me_Need_Ack = append(Me_Need_Ack, t.Data.Seq)
	}
	go_bootss.ChanSend <- t_Send

	var tt [][]election.NodeInfo
	for i := 0; i < len(Me_Need_Ack); i++ {
		select {
		case res := <-go_bootss.ChanMainNode:
			tt = append(tt, res)
		case <-time.After(Time_Out_Limit):
			log.Info("get main node failed")
		}
	}
	for i := 0; i < len(Me_Need_Ack); i++ {
		Is_in_Need_Ack(Me_Need_Ack[i])
	}
	if len(tt) < 2 {
		return 0, tt
	} else {
		return 1, tt
	}

}
func (this *bootss) Get_Block_Height(list_Ip_string []string) (int, []uint64) {

	var t_Send []p2p.Custsend
	var Me_Need_Ack []uint64
	for i := 0; i < len(list_Ip_string); i++ {
		t := Make_Send_Msg(LocalIP, list_Ip_string[i], true, 1, 0x0001)
		t_Send = append(t_Send, t)
		Need_Ack = append(Need_Ack, t.Data.Seq)
		Me_Need_Ack = append(Me_Need_Ack, t.Data.Seq)
	}
	go_bootss.ChanSend <- t_Send

	var tt []uint64
	for i := 0; i < 2; i++ {
		select {
		case res := <-go_bootss.ChanHeight:
			tt = append(tt, res.Len)
		case <-time.After(Time_Out_Limit):
			log.Info("BOOT", "get height failed")
		}
	}
	for i := 0; i < len(Me_Need_Ack); i++ {
		Is_in_Need_Ack(Me_Need_Ack[i])
	}

	if len(tt) < 2 {
		return 0, tt
	} else {
		return 1, tt
	}
}
func (this *bootss) init(s *eth.Ethereum) {
	this.Mine_Bc_Scheduler.P_blockchain = s.BlockChain()
	this.Mine_Bc_Scheduler.P_scheduler = s.Scheduler
	this.ChanSend = make(chan []p2p.Custsend, 100)
	this.ChanHeight = make(chan LocalHeightInfo)
	this.ChanMainNode = make(chan []election.NodeInfo)

}

func Touch_And_Analy_1(bb interface{}, cc p2p.Custsend) {
	Re_Analy_Data := cc
	switch Re_Analy_Data.Data.Type {
	case 0x0001:
		fmt.Println("0x0001")
		var t GetHeightRsp
		t.BlockHeight = go_bootss.Get_local_block_height()
		data, err := json.MarshalIndent(t, "", "   ")
		if err != nil {
			log.Info("BOOT", "json 0x0001 Marshaling failed")
		}
		Re_Analy_Data.Data.Data_struct = data
		Re_Analy_Data.Data.Type = 0x0002
		Re_Analy_Data.FromIp, Re_Analy_Data.ToIp = Re_Analy_Data.ToIp, Re_Analy_Data.FromIp

		var send []p2p.Custsend
		send = append(send, Re_Analy_Data)
		go_bootss.ChanSend <- send

	case 0x0002:
		fmt.Println("0x0002")
		if Is_in_Need_Ack(Re_Analy_Data.Data.Seq) != -1 {
			var aa LocalHeightInfo
			aa.Ip = Re_Analy_Data.FromIp
			var re_a GetHeightRsp
			if err := json.Unmarshal(Re_Analy_Data.Data.Data_struct, &re_a); err != nil {
				log.Info("BOOT", "Unmarshal failed err=", err)
			}
			aa.Len = re_a.BlockHeight
			go_bootss.ChanHeight <- aa
		} else {
			fmt.Println("BOOT", "0x0002 overtime height comming")
		}

	case 0x0003:
		fmt.Println("0x0003")
		var t GetMainNodeRsp
		t.MainNode = go_bootss.Get_Mine_Main_Node()
		data, err := json.MarshalIndent(t, "", "   ")
		if err != nil {
			log.Info("BOOT", "json 0x0003 Marshaling failed")
		}
		Re_Analy_Data.Data.Data_struct = data
		Re_Analy_Data.Data.Type = 0x0004
		Re_Analy_Data.FromIp, Re_Analy_Data.ToIp = Re_Analy_Data.ToIp, Re_Analy_Data.FromIp

		var Send_Main_Node []p2p.Custsend
		Send_Main_Node = append(Send_Main_Node, Re_Analy_Data)
		go_bootss.ChanSend <- Send_Main_Node
	case 0x0004:
		fmt.Println("0x0004")
		if Is_in_Need_Ack(Re_Analy_Data.Data.Seq) != -1 {
			var aa_4 []election.NodeInfo
			var re_aa GetMainNodeRsp
			if err := json.Unmarshal(Re_Analy_Data.Data.Data_struct, &re_aa); err != nil {
				log.Info("BOOT", "Unmarshal failed err=", err)
			}
			aa_4 = re_aa.MainNode
			go_bootss.ChanMainNode <- aa_4
		} else {
			fmt.Println("BOOT", "0x0004 overtime MainNode comming")
		}

	}
}
func Analy_Boot_Node(boot_node []string) []Boot_Node_info {
	var rr []Boot_Node_info
	for i := 0; i < len(boot_node); i++ {
		start := 0
		end := 0
		for j := 0; j < len(boot_node[i]); j++ {
			if boot_node[i][j] == '@' {
				start = j
			}
			if boot_node[i][j] == ':' {
				end = j
			}
		}
		var t Boot_Node_info
		t.Nodeid = boot_node[i][8:start]
		t.Ip = boot_node[i][start+1 : end]
		rr = append(rr, t)
	}
	return rr
}

func NetSearch(s *eth.Ethereum) RE_boot {

	boot_node := params.MainnetBootnodes
	boot_ip := Analy_Boot_Node(boot_node)

	go_bootss.init(s)
	sp2p := p2p.Slove_Cycle{}
	sp2p.Register_boot(Touch_And_Analy_1, go_bootss)
	go go_bootss.Read_Chan()

	for {
		var err error
		LocalIP, err = getip()
		if err != nil {
			log.Error("BOOT", " get local ip err err=", err)
			time.Sleep(2 * time.Second)
			continue
		}

		if len(LocalIP) != 0 {
			break
		} else {
			log.Error("BOOT", " err local ip! ip=", LocalIP)
			time.Sleep(2 * time.Second)
		}
	}

	log.Info("BOOT", "localip=", LocalIP)
	flag_default_boot := -1
	for i := 0; i < len(boot_ip); i++ {
		if LocalIP == boot_ip[i].Ip {
			flag_default_boot = i
		}
	}
	if flag_default_boot != -1 {
		log.Info("BOOT", "---is default---")
		var need_find_ip []string
		for i := 0; i < 3; i++ {
			if i != flag_default_boot {
				need_find_ip = append(need_find_ip, boot_ip[i].Ip)
			}
		}
		is_all_zero := Default_Boot_Find_Two_Boot(need_find_ip)

		if is_all_zero == 1 {
			var Main_List [][]election.NodeInfo
			var Status_Re_Main_Node int
			for true {
				Status_Re_Main_Node, Main_List = go_bootss.Get_Main_Node(need_find_ip)
				if Status_Re_Main_Node == 1 {
					Is_All_List_Empty := 1
					for i := 0; i < len(Main_List); i++ {
						if len(Main_List[i]) != 0 {
							Is_All_List_Empty = 0
						}
					}
					if Is_All_List_Empty == 1 {
						boot_FUN_RE.Net_Flag = 0
					} else {
						boot_FUN_RE.Net_Flag = 0
					}
					break
				} else {
					log.Info("three node height all zero   to get mainlistnode")
					time.Sleep(Time_Out_Limit)
				}

			}

		} else {
			Attept_Check_Not_Big_Than_Me(need_find_ip)

		}
	} else {
		var need_find_ip_not_default []string
		for i := 0; i < len(boot_ip); i++ {
			need_find_ip_not_default = append(need_find_ip_not_default, boot_ip[i].Ip)
		}
		Not_Default_Boot(need_find_ip_not_default)

	}

	log.Info("BOOT", "boot_FUN_RE.Net_Flag", boot_FUN_RE.Net_Flag, "boot_FUN_RE.Height", boot_FUN_RE.Height, "boot_FUN_RE.Main_List", boot_FUN_RE.Main_List)

	return boot_FUN_RE

}
