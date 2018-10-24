package pdmqd

import (
	"PDMQ/server/waitGroup"
	"fmt"
	"os"
	"sync"
)

type Topic struct {
	topicName     string
	channelMap    map[string]*Channel
	memoryMsgChan chan *Message
	waitGroup     waitGroup.WaitGroupWrapper
	ctx           *context
	sync.RWMutex
}

func CreateTopic(topicName string, ctx *context) *Topic {
	t := &Topic{
		topicName:     topicName,
		channelMap:    make(map[string]*Channel),
		memoryMsgChan: make(chan *Message, ctx.pdmqd.config.MsgChanSize),
		ctx:           ctx,
	}
	t.waitGroup.Wrap(t.msgOutput)
	return t
}

func (t *Topic) msgOutput() {
	var msg *Message
	//var buf []byte
	var memoryMsgChan chan *Message
	fmt.Println(msg.CreateMessageId())
	os.Exit(1)
	//var channelArr []*Channel

	//将所有的channel  放到一个数组 用于后续发送消息  ？ 感觉没卵用
	/*	for _, channel := range t.channelMap {
			channelArr := append(channelArr, channel)
		}
		fmt.Printf("%s", channelArr)*/
	select {
	case msg = <-memoryMsgChan:
		//msg = RevertMessage(buf)
	}

}

func (pdmqd *PDMQD) GetTopic(topicName string) *Topic {
	pdmqd.RLock()
	topic, ok := pdmqd.topicMap[topicName]
	pdmqd.RUnlock()
	if ok {
		return topic
	}
	topic = CreateTopic(topicName, &context{pdmqd: pdmqd})
	return topic
}

func (topic *Topic) GetChannel(channelName string) *Channel {
	topic.Lock()
	channel, ok := topic.channelMap[channelName]
	if !ok {
		channel = CreateChannel(topic.topicName, channelName, topic.ctx)
	}
	topic.channelMap[channelName] = channel
	topic.Unlock()

	return channel
}
