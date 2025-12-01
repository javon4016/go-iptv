package dao

import (
	"encoding/json"
	"fmt"
	"go-iptv/dto"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var WS = &WSClient{}
var Lic dto.Lic

// -------------------- 数据结构 --------------------

// 固定请求结构体
type Request struct {
	Action string      `json:"a"`
	Data   interface{} `json:"d"`
}

// 固定响应结构体
type Response struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

// -------------------- WebSocket 客户端 --------------------

type WSClient struct {
	url    string
	conn   *websocket.Conn
	lock   sync.Mutex
	done   chan struct{}
	closed bool
	retry  int
	count  int
}

// -------------------- 连接管理 --------------------

// 创建连接（带自动重连）
func ConLicense(url string) (*WSClient, error) {
	client := &WSClient{
		url:   url,
		done:  make(chan struct{}),
		retry: 5, // 最大重试次数
	}

	if err := client.connect(); err != nil {
		return nil, err
	}

	// 启动心跳检测
	go client.heartbeat()

	return client, nil
}

func (c *WSClient) connect() error {
	var err error
	for i := 1; i <= c.retry; i++ {
		dialer := websocket.Dialer{
			HandshakeTimeout:  5 * time.Second,
			EnableCompression: true,
		}
		c.conn, _, err = dialer.Dial(c.url, nil)
		if err == nil {
			c.count = 0
			log.Println("✅ 引擎连接成功")
			return nil
		}
		time.Sleep(5 * time.Second)
	}
	c.count++
	log.Printf("❌ 第 %d 次连接失败: %v, 3 秒后重试...", c.count, err)
	if c.count > 3 {
		c.count = 0
		return fmt.Errorf("❌ 多次连接失败，请检查引擎状态: %w", err)
	}
	c.connect()
	return fmt.Errorf("连接失败: %w", err)
}

// 判断 WS 是否已连接
func (c *WSClient) IsOnline() bool {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.conn != nil && !c.closed
}

// -------------------- 重启并重新连接 --------------------

// RestartLicense 会尝试重启 License 服务并重新建立 WS 连接

// -------------------- 心跳机制 --------------------

func (c *WSClient) heartbeat() {
	log.Println("启动心跳检测...")
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.lock.Lock()
			if c.closed || c.conn == nil {
				c.lock.Unlock()
				return
			}
			err := c.conn.WriteMessage(websocket.PingMessage, []byte("ping"))
			c.lock.Unlock()

			if err != nil {
				log.Println("⚠️ 心跳失败，尝试重连...")
				c.reconnect()
			}
		case <-c.done:
			return
		}
	}
}

// -------------------- 重连逻辑 --------------------

func (c *WSClient) reconnect() {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.closed {
		return
	}
	if c.conn != nil {
		c.conn.Close()
	}

	log.Println("🔄 尝试重连中...")
	if err := c.connect(); err != nil {
		log.Println("❌ 重连失败:", err)
	} else {
		log.Println("✅ 重连成功")
	}
}

// -------------------- 消息交互 --------------------

// 发送 JSON 并接收响应
func (c *WSClient) SendWS(req Request) (Response, error) {
	if !c.IsOnline() {
		return Response{}, fmt.Errorf("引擎未启动")
	}
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.closed {
		return Response{}, fmt.Errorf("连接已关闭")
	}

	// 发送
	if err := c.conn.WriteJSON(req); err != nil {
		log.Println("⚠️ 写入失败:", err)
		go c.reconnect()
		return Response{}, err
	}

	// 接收
	_, msg, err := c.conn.ReadMessage()
	if err != nil {
		log.Println("⚠️ 读取失败:", err)
		go c.reconnect()
		return Response{}, err
	}

	// 解析

	var resp Response
	if err := json.Unmarshal(msg, &resp); err != nil {
		return Response{}, fmt.Errorf("解析 JSON 失败: %w", err)
	}

	return resp, nil
}

// -------------------- 关闭连接 --------------------

func (c *WSClient) Close() {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.closed {
		return
	}

	c.closed = true
	close(c.done)

	if c.conn != nil {
		c.conn.Close()
		log.Println("🔒 引擎断开")
	}
}

// -------------------- 使用示例 --------------------

// func main() {
// 	url := "ws://127.0.0.1:8080/ws"

// 	client, err := ConnectWebSocket(url)
// 	if err != nil {
// 		log.Fatal("连接失败:", err)
// 	}
// 	defer client.Close()

// 	for {
// 		req := Request{
// 			Action: "echo",
// 			// Data:   map[string]any{"msg": "hello"},
// 		}

// 		resp, err := client.SendWS(req)
// 		if err != nil {
// 			log.Println("发送失败:", err)
// 			time.Sleep(2 * time.Second)
// 			continue
// 		}

// 		log.Println("响应: %+v\n", resp)
// 		time.Sleep(10 * time.Second)
// 	}
// }
