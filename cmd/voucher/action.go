package main

import (
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	logger "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

// amount of times to retry connecting to a disconnected node
var retries = 3

// amount of time to wait between retries
var retryTime = 4 * time.Second

func action(ctx *cli.Context) error {

	// check arguments
	if arguments := ctx.Args(); len(arguments) > 0 {
		return fmt.Errorf("failed to read command argument: %q", arguments[0])
	}

	if logLevel := ctx.GlobalString(LogLevelFlag.Name); logLevel != "" {
		log.WithField("logLevel", logLevel).Info("will configure log level")

		var err error
		log.Level, err = logger.ParseLevel(logLevel)
		if err != nil {
			log.WithError(err).Fatal("could not parse logLevel")
		}
	}

	//TODO: should we be binding all interfaces here ?
	//nolint:gosec
	ln, err := net.Listen("tcp", ":8081")
	if err != nil {
		log.Fatal(err)
	}

	log.Info("Voucher seeder up: accepting connections on port 8081")
	nodes := New()

	for {

		conn, ip := connectToNode(nodes, ln)
		if conn != nil {
			handleConnection(nodes, conn, ip)
		}

	}

}

func sendChallenge(conn net.Conn) (string, error) {
	challenge, err := GenerateRandomString(20)
	if err != nil {
		log.Fatal(err)
	}

	_, err = conn.Write([]byte(challenge + "\n"))
	return challenge, err
}

func getPort(rep []string) string {
	var port string
	if len(rep) == 1 {
		port = "8081"
	} else {
		port = strings.Split(rep[1], "\n")[0]
	}

	return port
}

func handleConnection(duskNode *DuskNodes, conn net.Conn, ip string) {
	// Issue challenge
	challenge, err := sendChallenge(conn)
	if err != nil {
		log.WithError(err).Error("could not process sendChallenge")
		_ = conn.Close()
		return
	}

	// read response
	buf := make([]byte, 256)
	if _, err := conn.Read(buf); err != nil {
		log.WithError(err).Error("could not read response")
		_ = conn.Close()
		return
	}

	log.WithField("message", string(buf)).Info("Message Received")

	rep := strings.Split(string(buf), ",")
	port := getPort(rep)
	ipPort := ip + ":" + port
	if HashesMatch(strings.TrimSpace(rep[0]), challenge) {
		// He's a good guy, add him to the node list and provide a list of connected nodes
		duskNode.Update(ipPort)
		ret := duskNode.DumpNodes(ipPort)
		if ret == "" {
			ret = "noip"
		}
		if _, err := conn.Write([]byte(ret)); err != nil {
			log.WithError(err).Error("could not write back to node")
			_ = conn.Close()
			return
		}
	} else {
		// Response fail, blacklist the node's ip
		duskNode.BlackList(ipPort, true)
		_ = conn.Close()
		return
	}

	// Keep reading until we get an EOF (connection closed)
	go func() {
		for {
			buf = make([]byte, 1)
			_ = conn.SetReadDeadline(time.Now().Add(retryTime))
			if _, err := conn.Read(buf); err == io.EOF {
				newConn := retryConnection(ipPort, retries)
				if newConn == nil {
					log.WithField("ipPort", ipPort).Warn("has disconnected")
					duskNode.SetInactive(ipPort)
					_ = conn.Close()
					return
				}
				_ = conn.Close()
				conn = newConn
			}
		}
	}()
}

// attempt to reconnect to a disconnected node
func retryConnection(ip string, retries int) net.Conn {
	for i := 0; i < retries; i++ {
		time.Sleep(retryTime)
		conn, err := net.Dial("tcp", ip)
		if err != nil {
			continue
		}
		return conn
	}
	return nil
}

func connectToNode(nodes *DuskNodes, ln net.Listener) (net.Conn, string) {
	conn, err := ln.Accept()
	if err != nil {
		log.Fatal(err)
	}
	ip, _, err := net.SplitHostPort(conn.RemoteAddr().String())
	if err != nil {
		log.Fatal(err)
	}
	log.WithField("ip", ip).Info("Incoming Connection")

	// Check that the node is not blacklisted, if it is then drop the connection
	if nodes.IsBlackListed(ip) {
		// Check if we can accept him
		unBAN, e := nodes.CheckUnBAN(ip)
		if e != nil {
			log.Fatal(e)
		}

		if !unBAN {
			_ = conn.Close()
			return nil, ""
		}
	}

	return conn, ip
}
