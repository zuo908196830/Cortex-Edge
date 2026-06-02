package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/hooks/auth"
	"github.com/mochi-mqtt/server/v2/listeners"
)

func main() {
	// 创建服务器
	server := mqtt.New(nil)

	// ✅ 必须添加这一行！允许所有连接（开发环境用）
	_ = server.AddHook(new(auth.AllowHook), nil)

	// 启动 TCP 监听器
	tcp := listeners.NewTCP(listeners.Config{
		ID:      "tcp1",
		Address: ":1883",
	})
	if err := server.AddListener(tcp); err != nil {
		log.Fatal(err)
	}

	// 启动服务
	go func() {
		if err := server.Serve(); err != nil {
			log.Fatal(err)
		}
	}()
	log.Println("MQTT Broker 启动成功，端口 1883")

	// 等待退出
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	_ = server.Close()
	log.Println("Broker 已关闭")
}
