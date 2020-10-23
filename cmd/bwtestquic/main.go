// Downloads torrents from the command-line.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/anacrolix/tagflag"
	"github.com/anacrolix/torrent/scion_torrent"
	log "github.com/inconshreveable/log15"
	"github.com/lucas-clemente/quic-go"
	"github.com/netsec-ethz/scion-apps/pkg/appnet"
	"github.com/netsec-ethz/scion-apps/pkg/appnet/appquic"
	"github.com/scionproto/scion/go/lib/snet"
	// "github.com/netsec-ethz/scion-apps/pkg/appnet"
)

var flags = struct {
	IsServer bool

	tagflag.StartPos
}{
	IsServer: true,
}

const (
	PacketSize int64 = 9000
	NumPackets int64 = 200000
)

func LogFatal(msg string, a ...interface{}) {
	log.Crit(msg, a...)
	os.Exit(1)
}

func Check(e error) {
	if e != nil {
		LogFatal("Fatal error. Exiting.", "err", e)
	}
}

func main() {
	if err := mainErr(); err != nil {
		log.Info("error in main: %v", err)
		os.Exit(1)
	}
}

func mainErr() error {
	tagflag.Parse(&flags)
	err := scion_torrent.InitSQUICCerts()
	Check(err)
	if flags.IsServer {
		runServer(42422)
	} else {
		addrStr := "19-ffaa:1:c3f,[127.0.0.1]:42422"
		serverAddr, err := appnet.ResolveUDPAddr(addrStr)
		Check(err)
		runClient(serverAddr)
	}

	return nil
}

func runClient(serverAddr *snet.UDPAddr) {
	// DCConn, err := appnet.DialAddr(serverAddr)
	sess, err := appquic.DialAddr(serverAddr, "127.0.0.1:42425", scion_torrent.TLSCfg, &quic.Config{
		KeepAlive: true,
	})
	Check(err)

	conn, sessErr := sess.OpenStreamSync(context.Background())
	Check(sessErr)
	// clientCCAddr := CCConn.LocalAddr().(*net.UDPAddr)
	sb := make([]byte, PacketSize)
	// Data channel connection
	// DCConn, err := appnet.DefNetwork().Dial(
	//	context.TODO(), "udp", clientCCAddr, serverAddr, addr.SvcNone)
	// Check(err)
	var i int64 = 0
	start := time.Now()
	for i < NumPackets {
		// Compute how long to wait

		// PrgFill(bwp.PrgKey, int(i*bwp.PacketSize), sb)
		// Place packet number at the beginning of the packet, overwriting some PRG data
		_, err := conn.Write(sb)
		Check(err)
		i++
	}
	elapsed := time.Since(start)
	fmt.Printf("Binomial took %s\n", elapsed)
}

func runServer(port uint16) error {

	conn, err := appnet.ListenPort(port)
	Check(err)

	qConn, listenErr := quic.Listen(conn, scion_torrent.TLSCfg, &quic.Config{KeepAlive: true})
	Check(listenErr)

	var numPacketsReceived int64
	numPacketsReceived = 0
	recBuf := make([]byte, PacketSize+1000)
	go func() {
		time.Sleep(5 * time.Second)
		fmt.Printf("Received %d packets\n", numPacketsReceived)
	}()
	x, err := qConn.Accept(context.Background())
	Check(err)
	DCConn, err := x.AcceptStream(context.Background())
	Check(err)
	for numPacketsReceived < NumPackets {
		n, err := DCConn.Read(recBuf)

		// Ignore errors, todo: detect type of error and quit if it was because of a SetReadDeadline
		if err != nil {
			fmt.Println(err)
			continue
		}
		if int64(n) != PacketSize {
			// The packet has incorrect size, do not count as a correct packet
			fmt.Println("Incorrect size.", n, "bytes instead of", PacketSize)
			continue
		}
		// fmt.Printf("Read packet of size %d\n", n)
		numPacketsReceived++
		// fmt.Printf("Received %d packets\n", numPacketsReceived)
	}

	fmt.Printf("Received all packets")
	return nil
}