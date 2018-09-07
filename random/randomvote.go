package random

import (
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/mc"
)

type RandomVote struct {
	roleUpdateCh  chan *mc.RoleUpdatedMsg
	roleUpdateSub event.Subscription

	currentRole      common.RoleType
	privatekey       *big.Int
	privatekeyHeight uint64
	msgcenter        *mc.Center
}

func newRandomVote(msgcenter *mc.Center) (*RandomVote, error) {

	randomvote := &RandomVote{
		roleUpdateCh:     make(chan *mc.RoleUpdatedMsg, 10),
		currentRole:      common.RoleDefault,
		privatekey:       big.NewInt(0),
		privatekeyHeight: 0,
		msgcenter:        msgcenter,
	}
	var err error
	randomvote.roleUpdateSub, err = mc.SubscribeEvent(mc.CA_RoleUpdated, randomvote.roleUpdateCh)
	if err != nil {
		return nil, err
	}
	go randomvote.update()
	return randomvote, nil
}

func (self *RandomVote) update() {
	defer self.roleUpdateSub.Unsubscribe()

	for {
		select {
		case RoleUpdateData := <-self.roleUpdateCh:
			log.INFO(ModuleVote, "RoleUpdateData", RoleUpdateData)
			self.RoleUpdateMsgHandle(RoleUpdateData)
		}
	}
}

func (self *RandomVote) RoleUpdateMsgHandle(RoleUpdateData *mc.RoleUpdatedMsg) error {
	self.currentRole = RoleUpdateData.Role
	if self.currentRole != common.RoleValidator {
		log.INFO(ModuleVote, "RoleUpdateMsgHandle", "当前不是验证者,忽略")
		return nil
	}

	height := RoleUpdateData.BlockNum
	if (height+5)%(common.GetBroadcastInterval()) != 0 {
		log.INFO(ModuleVote, "RoleUpdateMsgHandle", "当前不是投票点,忽略")
		return nil
	}

	privatekey, publickeySend, err := getkey()
	privatekeySend := common.BigToHash(privatekey).Bytes()
	if err != nil {
		return err
	}

	//TODO：生成公私钥
	/*
		与叶营沟通如下：
			1.调用生成公私钥方法还没确定，或者消息事件订阅，或者直接调用
	*/
	log.INFO(ModuleVote, "公钥 高度", (height + 10), "publickey", publickeySend)
	log.INFO(ModuleVote, "私钥 高度", (height + 10), "privatekey", privatekey, "privatekeySend", privatekeySend)
	mc.PublicEvent(mc.SendBroadCastTx, mc.BroadCastEvent{Txtyps: "SeedPublickey", Height: big.NewInt(int64(height)), Data: publickeySend})
	mc.PublicEvent(mc.SendBroadCastTx, mc.BroadCastEvent{Txtyps: "SeedPrivatekey", Height: big.NewInt(int64(height)), Data: privatekeySend})

	self.privatekey = privatekey
	self.privatekeyHeight = height
	return nil

}

func getkey() (*big.Int, []byte, error) {
	key, err := crypto.GenerateKey()
	if err != nil {
		return nil, nil, err
	}
	return key.D, keystore.ECDSAPKCompression(&key.PublicKey), err

}
