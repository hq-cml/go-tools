package network

import (
    "io"
    "log"
    "net"
)

const (
    KeepAlive     = "KEEP_ALIVE"
    NewConnection = "NEW_CONNECTION"
)

func CreateTCPListener(addr string) (*net.TCPListener, error) {
    tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
    if err != nil {
        return nil, err
    }
    tcpListener, err := net.ListenTCP("tcp", tcpAddr)
    if err != nil {
        return nil, err
    }
    return tcpListener, nil
}

func CreateTCPConn(addr string) (*net.TCPConn, error) {
    tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
    if err != nil {
        return nil, err
    }
    tcpConn, err := net.DialTCP("tcp",nil, tcpAddr)
    if err != nil {
        return nil, err
    }
    return tcpConn, nil
}

//建立Tcp连接中转
func Join2Conn(local *net.TCPConn, remote *net.TCPConn) {
    go joinConn(local, remote)
    go joinConn(remote, local)
}

func joinConn(conn1 *net.TCPConn, conn2 *net.TCPConn) {
    defer conn1.Close()
    defer conn2.Close()
    _, err := io.Copy(conn1, conn2)
    if err != nil {
        log.Println("copy failed ", err.Error())
        return
    }
}