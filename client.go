package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"time"

	"./protocol"
	"code.google.com/p/goprotobuf/proto"
	"github.com/spf13/cobra"
)

const MaximumSize = 1 << 16
const ReadTimeout = time.Second * 5

func getConn(address string) (*net.UDPConn, error) {
	addr, err := net.ResolveUDPAddr("udp4", address)
	if err != nil {
		return nil, err
	}
	conn, err := net.DialUDP("udp4", nil, addr)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func sendAndWait(conn *net.UDPConn, packet proto.Message) (*protocol.Packet, error) {
	var err error
	if err := send(conn, packet); err != nil {
		return nil, err
	}

	conn.SetReadDeadline(time.Now().Add(ReadTimeout))

	var n int
	incoming := make([]byte, MaximumSize)
	if n, err = conn.Read(incoming); err != nil {
		return nil, err
	}
	var incomingPacket protocol.Packet
	if err := proto.Unmarshal(incoming[0:n], &incomingPacket); err != nil {
		return nil, err
	}
	return &incomingPacket, nil
}

func send(conn *net.UDPConn, packet proto.Message) error {
	buff, err := proto.Marshal(packet)
	if err != nil {
		return err
	}
	_, err = conn.Write(buff)
	return err
}

func main() {
	var address string

	cmd := &cobra.Command{
		Use: "msg-client",
	}
	cmd.PersistentFlags().StringVarP(&address, "address", "a", "localhost:8003", "server address")

	// Query
	cmd.AddCommand(&cobra.Command{
		Use:   "query <mailbox>",
		Short: "Query a list of messages in a mailbox",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 1 {
				fmt.Fprintf(os.Stderr, "error: missing mailbox\n")
				return
			}
			conn, err := getConn(address)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", err)
				return
			}
			defer conn.Close()

			packet := protocol.Packet{
				Type: protocol.Packet_Query.Enum(),
				Query: &protocol.Query{
					Mailbox: &args[0],
				},
			}
			resp, err := sendAndWait(conn, &packet)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", err)
				return
			}
			messageIds := resp.MessageIds
			if *resp.Type != protocol.Packet_MessageIds || messageIds == nil || messageIds.Ids == nil {
				fmt.Fprintf(os.Stderr, "no messages\n")
				return
			}
			for _, id := range messageIds.Ids {
				fmt.Printf("%s\n", id)
			}
		},
	})

	// Fetch
	cmd.AddCommand(&cobra.Command{
		Use:   "fetch <mailbox> <message ids...>",
		Short: "Fetch messages from a mailbox",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 1 {
				fmt.Fprintf(os.Stderr, "error: missing mailbox\n")
				return
			}
			if len(args) < 2 {
				fmt.Fprintf(os.Stderr, "error: missing message ids\n")
				return
			}
			conn, err := getConn(address)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", err)
				return
			}
			defer conn.Close()

			packet := protocol.Packet{
				Type: protocol.Packet_Fetch.Enum(),
				MessageIds: &protocol.MessageIds{
					Mailbox: &args[0],
					Ids:     args[1:],
				},
			}
			resp, err := sendAndWait(conn, &packet)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", err)
				return
			}
			messages := resp.Messages
			if messages == nil || messages.Messages == nil {
				fmt.Fprintf(os.Stderr, "error: no messages\n")
				return
			}
			files := []string{}
			for _, message := range messages.Messages {
				f, err := ioutil.TempFile(os.TempDir(), "message")
				if err != nil {
					fmt.Fprintf(os.Stderr, "%s\n", err)
					continue
				}
				fmt.Fprintf(f, "ID: %s\n", *message.Id)
				fmt.Fprintf(f, "Mailbox: %s\n", *message.Mailbox)
				fmt.Fprintf(f, "Sender: %s\n", *message.Sender)
				fmt.Fprintf(f, "Timestamp: %s\n", *message.Timestamp)
				fmt.Fprintf(f, "\n%s", *message.Body)
				f.Close()
				files = append(files, f.Name())
			}
			pager := exec.Command("less", files...)
			pager.Stdin = os.Stdin
			pager.Stdout = os.Stdout
			pager.Stderr = os.Stderr
			if err := pager.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "error: %s\n", err)
			}
			for _, file := range files {
				os.Remove(file)
			}
		},
	})

	// Send
	cmd.AddCommand(&cobra.Command{
		Use:   "send <mailbox>",
		Short: "Sends a message to a mailbox",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 1 {
				fmt.Fprintf(os.Stderr, "error: missing mailbox\n")
				return
			}
			conn, err := getConn(address)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", err)
				return
			}
			defer conn.Close()

			packet := protocol.Packet{
				Type:     protocol.Packet_Send.Enum(),
				Messages: &protocol.Messages{},
			}
			arr, err := ioutil.ReadAll(os.Stdin)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", err)
				return
			}
			str := string(arr)
			packet.Messages.Messages = []*protocol.Message{
				&protocol.Message{
					Id:        proto.String(""),
					Mailbox:   &args[0],
					Sender:    proto.String(""),
					Timestamp: proto.String(""),
					Body:      &str,
				},
			}
			if err := send(conn, &packet); err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", err)
				return
			}
		},
	})

	cmd.Execute()
}
