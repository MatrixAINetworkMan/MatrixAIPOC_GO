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
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/election"
	"encoding/json"
)

type voteResult struct {
	//State
	Result bool `json:"result"    gencodec:"required"`
	NodeId string `json:"nodeid"    gencodec:"required"`
}

func TestMsg(t *testing.T)  {
	var v, n voteResult
	v.Result = true
	v.NodeId = "dbf8dcc4c82eb2ea2e1350b"

	s,err:= json.MarshalIndent(v, "", "   ")
	fmt.Println(s)
	fmt.Println(v)
	fmt.Println(err)
	err = json.Unmarshal(s,  &n)
	fmt.Println(n)


}

func TestNotify(t *testing.T) {

	fmt.Println("start Test notify!!!")

	var returnList election.NodeList
	minerlist := election.NodeInfo{
		TPS: 10000,
		IP:  "192.168.3.124",
	}
	CommitteeList := election.NodeInfo{
		TPS: 20000,
		IP:  "127.0.0.1",
	}
	//bothlist := election.NodeInfo{
	//TPS: 30000,
	//}
	//offlinelist := election.NodeInfo{
	//TPS: 40000,
	//}
	returnList.MinerList = append(returnList.MinerList, minerlist)
	returnList.CommitteeList = append(returnList.CommitteeList, CommitteeList)
	//returnList.Both = append(returnList.Both, bothlist)
	//returnList.OfflineList = append(returnList.OfflineList, offlinelist)
	V := New(nil, nil)
	//V.Start(&returnList)
	V.Notify(uint64(99))
	time.Sleep(2 * time.Second)
	V.Stop()
}

func TestLeader(t *testing.T) {
	fmt.Println("start Test Leader !!!")
	//runtime.GOMAXPROCS(8)

	//go test_follower()
	//go test_leader()

	//leader := new(Verifier)
	//leader.role = 1
	//leader.Start()

	//follower := new(Verifier)
	//follower.role = 0
	//follower.Start()

	//test_stopLeader()
	//test_stopFollower()
	//time.Sleep(2 * time.Second)
}


var  leaderWorkCh chan uint8

func  timefinshfunc(status *uint8, timeout time.Duration){
	time.Sleep(timeout)
	leaderWorkCh <- TIMEIN
}

func  timeoutfunc(status *uint8, timeout time.Duration){
	time.Sleep(timeout)
	if TIMEUNKNOW==*status{
		leaderWorkCh <- TIMEOUT
	} else {
		return
	}
}

func TestLeaderTimeOutNotify(t *testing.T) {
	leaderWorkCh = make(chan uint8,1)
	fmt.Println("values:", 0)
	var pretimeout *uint8
	leaderWorkCh <-TIMEUNKNOW

	for{
		select {
		case status := <-leaderWorkCh:
			timeout := new(uint8)
			fmt.Println(time.Now(),"leaderWorkCh")
			//赋初值
			*timeout = TIMEUNKNOW
			//处理超时或完成
			if TIMEOUT == status {
				fmt.Println(time.Now(),"pre timeout:")
			} else if TIMEIN == status{
				*pretimeout = TIMEIN//改变前置
				fmt.Println(time.Now(),"pre finish:")
			}

			fmt.Println("run:")
			// for test
			go timeoutfunc(timeout,2*time.Second)
			pretimeout = timeout
			fmt.Println(time.Now(),"pretimeout = timeout:")
		}
	}
}




func TestLeaderTimeFinishNotify(t *testing.T) {
	leaderWorkCh = make(chan uint8,1)
	fmt.Println("values:", 0)
	var pretimeout *uint8
	leaderWorkCh <-TIMEUNKNOW

	for{
		select {
		case status := <-leaderWorkCh:
			timeout := new(uint8)
			fmt.Println(time.Now(),"leaderWorkCh")
			//赋初值
			*timeout = TIMEUNKNOW
			//处理超时或完成
			if TIMEOUT == status {
				fmt.Println(time.Now(),"pre timeout:")
			} else if TIMEIN == status{
				*pretimeout = TIMEIN//改变前置
				fmt.Println(time.Now(),"pre finish:")
			}

			fmt.Println("run:")
			// for test
			go timefinshfunc(timeout,1*time.Second)
			go timeoutfunc(timeout,2*time.Second)
			pretimeout = timeout
			fmt.Println(time.Now(),"pretimeout = timeout:")
		}
	}
}


func TestLeaderTimeRandNotify(t *testing.T) {
	leaderWorkCh = make(chan uint8,1)
	fmt.Println("values:", 0)
	var pretimeout *uint8
	leaderWorkCh <-TIMEUNKNOW
	i:= 0
	for{
		select {
		case status := <-leaderWorkCh:
			timeout := new(uint8)
			fmt.Println(time.Now(),"leaderWorkCh")
			//赋初值
			*timeout = TIMEUNKNOW
			//处理超时或完成
			if TIMEOUT == status {
				fmt.Println(time.Now(),"pre timeout:")
			} else if TIMEIN == status{
				*pretimeout = TIMEIN//改变前置
				fmt.Println(time.Now(),"pre finish:")
			}

			fmt.Println("run:")
			// for test
			if 1==i%2{
				fmt.Println(time.Now(),"go  timefinshfunc:",i)
				go timefinshfunc(timeout,1*time.Second)
			}
			i++


			go timeoutfunc(timeout,2*time.Second)
			pretimeout = timeout
			fmt.Println(time.Now(),"pretimeout = timeout:")
		}
	}
}
func TestDposTx(t *testing.T) {
	var v Verifier
	v.verifierList = make([]election.NodeInfo, 4)

	for i := range v.verifierList {
		v.verifierList[i] = election.NodeInfo{
			TPS:        uint32(i),
			IP:         "",
			ID:         string(i),
			Wealth:     uint64(i),
			OnlineTime: uint64(i),
			TxHash:     uint64(i),
			Value:      uint64(i),
		}
	}

	var voteResult []VoteResult
	voteResult = make([]VoteResult, 4)
	for i := 0; i < 2; i++ {
		voteResult[i] = VoteResult{result: true, nodeId: string(i)}
	}
	for i := 2; i < 4; i++ {
		voteResult[i] = VoteResult{result: false, nodeId: string(i)}
	}

	if result := v.DposTx(voteResult); result {
		t.Errorf("Expect false as this case only 2 of 4 passed")
	}
	voteResult[2].result = true
	if result := v.DposTx(voteResult); result {
		t.Error("Expect false as although 3 of 4 passed, but the value of valid node is only 75%")
	}
	voteResult[3].result = true
	if result := v.DposTx(voteResult); !result {
		t.Error("Expect true, as all node are valid and all results are true")
	}
}
