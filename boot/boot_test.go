package boot

import (
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/p2p"

	"github.com/ethereum/go-ethereum/core"
)

func TestNewBoot(t *testing.T) {
	go p2p.Receiveudp()
	go p2p.CustSend()
	var bc *core.BlockChain
	ans := New(bc, "asdddd")
	ans.Run()

	time.Sleep(100 * time.Second)

	fmt.Println(ans)
}
