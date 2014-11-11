package main

import (
	"database/sql"
	"flag"
	"fmt"
	"net"

	"./protocol"
	"code.google.com/p/goprotobuf/proto"
	_ "github.com/mattn/go-sqlite3"
)

const MaximumSize = 1 << 16

var db *sql.DB

func sendMessage(conn *net.UDPConn, addr *net.UDPAddr, packet proto.Message) error {
	buff, err := proto.Marshal(packet)
	if err != nil {
		return err
	}
	if _, err = conn.WriteToUDP(buff, addr); err != nil {
		return err
	}
	return nil
}

func handlePacket(conn *net.UDPConn, addr *net.UDPAddr, packet protocol.Packet) {
	// Query
	if *packet.Type == protocol.Packet_Query {
		query := packet.Query
		if query == nil {
			return
		}
		rows, err := db.Query(`SELECT id FROM messages WHERE mailbox = ?`, *query.Mailbox)
		if err != nil {
			return
		}
		defer rows.Close()
		reply := protocol.Packet{
			Type: protocol.Packet_MessageIds.Enum(),
			MessageIds: &protocol.MessageIds{
				Mailbox: query.Mailbox,
			},
		}
		for rows.Next() {
			var id string
			rows.Scan(&id)
			reply.MessageIds.Ids = append(reply.MessageIds.Ids, id)
		}
		sendMessage(conn, addr, &reply)
		return
	}

	// Fetch
	if *packet.Type == protocol.Packet_Fetch {
		messageIds := packet.MessageIds
		if messageIds == nil {
			return
		}
		reply := protocol.Packet{
			Type:     protocol.Packet_Fetch.Enum(),
			Messages: &protocol.Messages{},
		}
		mailbox := messageIds.Mailbox
		ids := messageIds.Ids
		if mailbox == nil || ids == nil {
			sendMessage(conn, addr, &reply)
			return
		}
		for _, id := range ids {
			row := db.QueryRow(`SELECT id, mailbox, sender, timestamp, body FROM messages WHERE mailbox = ? AND id = ? LIMIT 1`, *mailbox, id)
			if row == nil {
				continue
			}
			var id string
			var mailbox string
			var sender string
			var timestamp string
			var body string
			if err := row.Scan(&id, &mailbox, &sender, &timestamp, &body); err != nil {
				continue
			}
			message := protocol.Message{
				Id:        &id,
				Mailbox:   &mailbox,
				Sender:    &sender,
				Timestamp: &timestamp,
				Body:      &body,
			}
			reply.Messages.Messages = append(reply.Messages.Messages, &message)
		}
		sendMessage(conn, addr, &reply)
		return
	}

	// Send
	if *packet.Type == protocol.Packet_Send {
		messages := packet.Messages
		if messages == nil || messages.Messages == nil {
			return
		}
		messageSlice := messages.Messages
		if len(messageSlice) < 1 {
			return
		}
		sender := addr.IP.String()
		message := messageSlice[0]
		if _, err := db.Exec(`INSERT INTO messages (mailbox, sender, timestamp, body) VALUES (?, ?, datetime('now'), ?)`, *message.Mailbox, sender, *message.Body); err != nil {
			println(err.Error())
		}
		return
	}
}

func main() {
	sqliteFile := flag.String("sqlite", "./db.sqlite3", "sqlite3 database file")
	address := flag.String("address", ":8003", "address to bind the server to")
	flag.Parse()

	var err error
	db, err = sql.Open("sqlite3", *sqliteFile)
	if err != nil {
		fmt.Printf("%s\n", err)
		return
	}
	defer db.Close()

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS messages (
		  id        INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT UNIQUE,
		  mailbox   TEXT NOT NULL,
		  sender    TEXT NOT NULL,
		  timestamp TEXT NOT NULL,
		  body      TEXT)`); err != nil {
		fmt.Printf("%s\n", err)
		return
	}

	addr, err := net.ResolveUDPAddr("udp4", *address)
	if err != nil {
		fmt.Printf("%s\n", err)
		return
	}

	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		fmt.Printf("%s\n", err)
		return
	}
	fmt.Printf("Waiting for messages on %v\n", addr)

	buffer := make([]byte, MaximumSize)
	for {
		var packet protocol.Packet
		n, addr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			fmt.Printf("Read error: %s\n", err)
			continue
		}
		if err := proto.Unmarshal(buffer[0:n], &packet); err != nil {
			fmt.Printf("Unmarshal error: %s\n", err)
			continue
		}
		fmt.Printf("Received message from %v\n", addr)
		go handlePacket(conn, addr, packet)
	}
}
