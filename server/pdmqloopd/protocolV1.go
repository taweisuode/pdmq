/**
 * @Time : 2019-07-15 12:02
 * @Author : zhuangjingpeng
 * @File : protocolV1
 * @Desc : file function description
 */
package pdmqloopd

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cihub/seelog"
	"io"
	"log"
	"net"
	"os"
	"time"
)

type LoopMessage struct {
	MessageType []byte
	Body        []byte
}
type protocolV1 struct {
	ctx *context
}

func (p *protocolV1) IOLoop(connect net.Conn) error {
	var (
		err  error
		line []byte
	)
	client := NewClientV1(connect)
	reader := bufio.NewReader(client)

	//接收pdmqd 的 ping 消息 以及 consumer 的注册消息
	for {
		line, err = reader.ReadSlice('\n')
		if err != nil {
			if err == io.EOF {
				err = nil
			} else {
				seelog.Errorf("failed to read command - %s", err)
			}
			break
		}
		params := bytes.Split(line, []byte(" "))

		for _, b := range params {
			fmt.Printf("params is [%+v]\n", string(b))
		}

		fmt.Printf("[PDMQLOOPD] [%+v] get pdmqd request params: %+v\n", time.Now().Format("2006-01-02 15:04:05"), string(params[0]))

		//这里返回自己封装的msg 而不是nsq []byte
		msg, err := p.Exec(client, reader, params)

		fmt.Printf("receive msg is [%+v],err is [%+v]\n", string(msg.Body), err)
		if err != nil && msg == nil {
			seelog.Errorf("response is [%v],err is [%v]", msg, err)
			continue
		}
		if params[0] != nil && msg != nil {
			err = p.SendMessage(client, msg)
			if err != nil {
				seelog.Errorf("response is [%v],err is [%v]", msg, err)
				continue
			}
		}
	}
	connect.Close()
	//todo：链接断开 需要将对应的producer 删除
	return err
}

func (p *protocolV1) SendMessage(client *clientV1, msg *LoopMessage) error {

	fmt.Printf("[PDMQLOOPD] [%+v] msg MessageType is [%+v],msg body is [%+v]\n", time.Now().Format("2006-01-02 15:04:05"), string(msg.MessageType), string(msg.Body))
	err := p.Send(client, msg.Body)

	if err != nil {
		seelog.Errorf(" protocolV1 send error %v\n", err.Error())
		return err
	}
	return nil
}

func (p *protocolV1) Send(client *clientV1, buf []byte) error {

	_, err := p.SendProtocolResponse(client, buf)
	return err

}

func (p *protocolV1) Exec(client *clientV1, reader *bufio.Reader, params [][]byte) (*LoopMessage, error) {
	msg := &LoopMessage{
		MessageType: params[0],
		Body:        []byte("OK"),
	}
	switch {
	case bytes.Equal(bytes.Trim(params[0], "\n"), []byte("PING")):
		break
	case bytes.Equal(bytes.Trim(params[0], "\n"), []byte("IDENTIFY")):
		body, err := p.IDENTIFY(client, reader, params)
		msg.Body = body
		return msg, err
	case bytes.Equal(bytes.Trim(params[0], "\n"), []byte("REGISTER")):
		break
	case bytes.Equal(bytes.Trim(params[0], "\n"), []byte("UNREGISTER")):
		break
	}
	return msg, nil
}

//客户端注册时 调用sub 请求
//往SubEventChan中投递，而这个chan 在pdmqd中的messagePush 中 接收
func (p *protocolV1) IDENTIFY(client *clientV1, reader *bufio.Reader, params [][]byte) ([]byte, error) {
	var err error
	if client.peerInfo != nil {
		return nil, errors.New("cannot IDENTIFY again")
	}
	var bodyLen int32
	err = binary.Read(reader, binary.BigEndian, &bodyLen)
	fmt.Printf("bodyLen is [%+v]\n", bodyLen)
	if err != nil {
		return nil, errors.New("IDENTIFY failed to read body size")
	}
	body := make([]byte, bodyLen)
	_, err = io.ReadFull(reader, body)
	if err != nil {
		return nil, errors.New("IDENTIFY failed to read body")
	}

	peerInfo := PeerInfo{id: client.RemoteAddr().String()}
	err = json.Unmarshal(body, &peerInfo)

	peerInfo.RemoteAddress = client.RemoteAddr().String()
	client.peerInfo = &peerInfo

	fmt.Printf("peerInfo is [%+v]\n", client.peerInfo)

	//这里封装成peerinfo 结构体返回信息
	// build a response
	data := make(map[string]interface{})
	data["tcp_port"] = p.ctx.pdmqloopd.RealTCPAddr().Port
	data["http_port"] = p.ctx.pdmqloopd.RealHTTPAddr().Port
	data["version"] = p.ctx.pdmqloopd.version
	hostname, err := os.Hostname()
	if err != nil {
		log.Fatalf("ERROR: unable to get hostname %s", err)
	}
	data["broadcast_address"] = hostname
	data["hostname"] = hostname

	response, err := json.Marshal(data)

	fmt.Printf("response is [%+v]\n", string(response))
	if err != nil {
		return []byte("OK"), nil
	}
	return response, nil
}

//Todo：这一版不做处理，需要为消费端限流做反馈
func (p *protocolV1) RDY(client *clientV1, params [][]byte) error {
	return nil
}

//发送该协议统一的返回信息
func (p *protocolV1) SendProtocolResponse(w io.Writer, data []byte) (int, error) {
	n, err := w.Write(data)

	fmt.Printf("write to client data is [%+v], len is [%d]\n", string(data), n)
	seelog.Infof("write to client data is [%+v], len is [%d]\n", string(data), n)

	return n, err
}
