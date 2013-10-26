package zmqt

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/goraft/raft"
	zmq "github.com/pebbe/zmq4"
)

var logger *log.Logger

func init() {
	logger = log.New(os.Stdout, "[zmqt]", log.Lmicroseconds)
}

// ZMQ transporter is a transport layer used for communication
// between multiple servers using ZMQ messaging.
type ZmqTransporter struct {
}

var (
	MSG_VOTE          = "vote"
	MSG_APPENDENTRIES = "appendEntries"
)

func NewZmqTransporter() *ZmqTransporter {
	tr := &ZmqTransporter{}
	return tr
}
func (t *ZmqTransporter) SendVoteRequest(server raft.Server, peer *raft.Peer, req *raft.RequestVoteRequest) *raft.RequestVoteResponse {
	socket, _ := zmq.NewSocket(zmq.REQ)
	traceln("connecting to peer:", peer.ConnectionString)
	socket.Connect(peer.ConnectionString)
	defer socket.Close()

	var b bytes.Buffer
	if _, err := req.Encode(&b); err != nil {
		traceln("transporter.rv.encoding.error:", err)
		return nil
	}
	socket.SendMessage(MSG_VOTE, b.Bytes())
	// response handling
	poller := zmq.NewPoller()
	poller.Add(socket, zmq.POLLIN)

	events, err := poller.Poll(time.Second)
	if err != nil || len(events) == 0 {
		traceln("transporter.rv.encoding.error:", err)
		return nil
	}

	response, err := socket.RecvBytes(0)
	resp := &raft.RequestVoteResponse{}
	if _, err = resp.Decode(bytes.NewReader(response)); err != nil && err != io.EOF {
		traceln("transporter.ae.decoding.error:", err)
		return nil
	}
	return resp
}

func (t *ZmqTransporter) SendAppendEntriesRequest(server raft.Server, peer *raft.Peer, req *raft.AppendEntriesRequest) *raft.AppendEntriesResponse {
	socket, _ := zmq.NewSocket(zmq.REQ)
	traceln("connecting to peer:", peer.ConnectionString)
	socket.Connect(peer.ConnectionString)
	defer socket.Close()

	var b bytes.Buffer
	if _, err := req.Encode(&b); err != nil {
		traceln("transporter.rv.encoding.error:", err)
		return nil
	}
	socket.SendMessage(MSG_APPENDENTRIES, b.Bytes())
	// response handling
	poller := zmq.NewPoller()
	poller.Add(socket, zmq.POLLIN)

	events, err := poller.Poll(time.Second)
	if err != nil || len(events) == 0 {
		traceln("transporter.rv.encoding.error:", err)
		return nil
	}
	response, err := socket.RecvBytes(0)
	resp := &raft.AppendEntriesResponse{}
	if _, err = resp.Decode(bytes.NewReader(response)); err != nil && err != io.EOF {
		traceln("transporter.ae.decoding.error:", err)
		return nil
	}
	return resp
}

// Sends a SnapshotRequest RPC to a peer.
func (t *ZmqTransporter) SendSnapshotRequest(server raft.Server, peer *raft.Peer, req *raft.SnapshotRequest) *raft.SnapshotResponse {
	return nil
}

// Sends a SnapshotRequest RPC to a peer.
func (t *ZmqTransporter) SendSnapshotRecoveryRequest(server raft.Server, peer *raft.Peer, req *raft.SnapshotRecoveryRequest) *raft.SnapshotRecoveryResponse {
	return nil
}

func (t *ZmqTransporter) Install(bindto string, server raft.Server) {
	traceln("Initializing ZMQ socket, binding to:", bindto)
	go func() {
		socket, _ := zmq.NewSocket(zmq.REP)
		socket.Bind(bindto)
		defer socket.Close()
	LOOP:
		for {
			msg, err := socket.RecvMessageBytes(0)
			if err != nil {
				traceln("transporter.rcvmsg:", err)
				break LOOP // signal received, exit
			}
			switch string(msg[0]) {
			case MSG_VOTE:
				req := &raft.RequestVoteRequest{}
				if _, err := req.Decode(bytes.NewReader(msg[1])); err != nil {
					socket.Send(fmt.Sprintf("%v", err), 0)
					continue
				}

				resp := server.RequestVote(req)
				response := new(bytes.Buffer)
				if _, err := resp.Encode(response); err != nil {
					socket.Send(fmt.Sprintf("%v", err), 0)
					continue
				}
				socket.SendBytes(response.Bytes(), 0)

			case MSG_APPENDENTRIES:
				req := &raft.AppendEntriesRequest{}
				if _, err := req.Decode(bytes.NewReader(msg[1])); err != nil {
					socket.Send(fmt.Sprintf("%v", err), 0)
					continue
				}

				resp := server.AppendEntries(req)
				response := new(bytes.Buffer)
				if _, err := resp.Encode(response); err != nil {
					socket.Send(fmt.Sprintf("%v", err), 0)
					continue
				}
				socket.SendBytes(response.Bytes(), 0)
			}
		}
	}()
}

// Prints to the standard logger if trace debugging is enabled. Arguments
// are handled in the manner of debugln.
func traceln(v ...interface{}) {
	if raft.LogLevel() >= raft.Trace {
		logger.Println(v...)
	}
}
