package main

import (
    "bufio"
    "fmt"
    "io"
    "log"
    "net"

    "github.com/hq-cml/go-tools/net-across/network"
)

var (
    // 本地内网服务端口
    localServerAddr = "127.0.0.1:81"

    // 远端的服务控制通道，用来传递控制信息，如出现新连接和心跳
    //remoteControlAddr = "111.111.111.111:8009"
    remoteControlAddr = "127.0.0.1:8009"

    // 远端服务端口，用来建立隧道
    //remoteServerAddr  = "111.111.111.111:8008"
    remoteServerAddr  = "127.0.0.1:8008"
)

func main() {
    //和远端服务建立控制连接
    tcpConn, err := network.CreateTCPConn(remoteControlAddr)
    if err != nil {
        log.Println("[连接失败]" + remoteControlAddr + err.Error())
        return
    }
    log.Println("[已连接]" + remoteControlAddr)

    reader := bufio.NewReader(tcpConn)
    for {
        s, err := reader.ReadString('\n')
        if err != nil || err == io.EOF {
            break
        }

        fmt.Println("Recv ControlMsg:", s)

        // 当有新连接信号出现时，新建一个tcp连接
        if s == network.NewConnection + "\n" {
            go connectLocalAndRemote()
        }
    }

    log.Println("[已断开]" + remoteControlAddr)
}

func connectLocalAndRemote() {
    local := connectLocal()
    remote := connectRemote()

    if local != nil && remote != nil {
        //建立tcp连接中转
        network.Join2Conn(local, remote)
    } else {
        //异常处理
        if local != nil {
            _ = local.Close()
        }
        if remote != nil {
            _ = remote.Close()
        }
    }
}

//建立和本地内网服务的连接
func connectLocal() *net.TCPConn {
    conn, err := network.CreateTCPConn(localServerAddr)
    if err != nil {
        log.Println("[连接本地服务失败]" + err.Error())
    }
    return conn
}

//建立和远端的隧道连接
func connectRemote() *net.TCPConn {
    conn, err := network.CreateTCPConn(remoteServerAddr)
    if err != nil {
        log.Println("[连接远端服务失败]" + err.Error())
    }
    return conn
}