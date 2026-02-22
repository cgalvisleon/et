package tcp

import (
	"math/rand"
	"slices"
	"sync"
	"time"

	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/timezone"
)

var (
	rngMu             sync.Mutex
	rng               = rand.New(rand.NewSource(time.Now().UnixNano()))
	heartbeatInterval = 500 * time.Millisecond
)

/**
* randomBetween
* @param minMs, maxMs int
* @return time.Duration
**/
func randomBetween(minMs, maxMs int) time.Duration {
	if minMs >= maxMs {
		return time.Duration(minMs) * time.Millisecond
	}

	rngMu.Lock()
	n := rng.Intn(maxMs-minMs+1) + minMs
	rngMu.Unlock()

	return time.Duration(n) * time.Millisecond
}

/**
* majority
* @param n int
* @return int
**/
func majority(n int) int {
	return (n / 2) + 1
}

type ResponseBool struct {
	Ok    bool
	Error error
}

type RequestVoteArgs struct {
	Term        int
	CandidateID string
}

type RequestVoteReply struct {
	Term        int
	VoteGranted bool
}

type HeartbeatArgs struct {
	Term     int
	LeaderID string
}

type HeartbeatReply struct {
	Term int
	Ok   bool
}

type Raft struct {
	server        *Server                `json:"-"`
	addr          string                 `json:"-"`
	registry      map[string]HandlerFunc `json:"-"`
	peers         []*Client              `json:"-"`
	state         Mode                   `json:"-"`
	term          int                    `json:"-"`
	votedFor      string                 `json:"-"`
	leaderID      string                 `json:"-"`
	lastHeartbeat time.Time              `json:"-"`
	turn          int                    `json:"-"`
	muRaft        sync.Mutex             `json:"-"`
	muTurn        sync.Mutex             `json:"-"`
	RequestVote   HandlerFunc
	Heartbeat     HandlerFunc
}

/**
* build
* @return map[string]HandlerFunc
**/
func (s *Raft) build() map[string]HandlerFunc {
	s.registry = map[string]HandlerFunc{
		"RequestVote": s.RequestVote,
		"Heartbeat":   s.Heartbeat,
	}
	return s.registry
}

/**
* addNode
* @param addr string
**/
func (s *Raft) addNode(addr string) {
	if s.addr == addr {
		return
	}

	node := NewNode(addr)
	s.peers = append(s.peers, node)
}

/**
* removeNode
* @param addr string
**/
func (s *Raft) removeNode(addr string) {
	idx := slices.IndexFunc(s.peers, func(e *Client) bool { return e.Addr == addr })
	if idx != -1 {
		s.peers = append(s.peers[:idx], s.peers[idx+1:]...)
	}
}

/**
* LeaderID
* @return string, bool
**/
func (s *Raft) LeaderID() (leader string, imLeader bool) {
	s.muRaft.Lock()
	leader = s.leaderID
	state := s.state
	s.muRaft.Unlock()
	imLeader = state == Leader
	return
}

/**
* getLeader
* @return *Client, bool
**/
func (s *Raft) getLeader() (*Client, bool) {
	leader, imLeader := s.LeaderID()
	if imLeader {
		return nil, true
	}

	idx := slices.IndexFunc(s.peers, func(e *Client) bool { return e.Addr == leader })
	if idx == -1 {
		return nil, false
	}

	return s.peers[idx], false
}

/**
* nextTurn
* @return *Client
**/
func (s *Raft) nextTurn() *Client {
	s.muTurn.Lock()
	result := s.peers[s.turn]
	s.turn++
	s.muTurn.Unlock()

	return result
}

/**
* electionLoop
**/
func (s *Raft) electionLoop() {
	if len(s.peers) == 0 {
		s.muRaft.Lock()
		s.becomeLeader()
		s.muRaft.Unlock()
		return
	}

	s.muRaft.Lock()
	s.state = Follower
	s.lastHeartbeat = timezone.Now()
	s.muRaft.Unlock()

	for {
		timeout := randomBetween(1500, 3000)
		time.Sleep(timeout)

		s.muRaft.Lock()
		elapsed := time.Since(s.lastHeartbeat)
		state := s.state
		s.muRaft.Unlock()

		if elapsed > heartbeatInterval && state != Leader {
			s.startElection()
		}
	}
}

/**
* startElection
**/
func (s *Raft) startElection() {
	s.muRaft.Lock()
	s.state = Candidate
	s.term++
	term := s.term
	s.votedFor = s.addr
	s.muRaft.Unlock()

	votes := 1
	total := len(s.peers)
	for _, peer := range s.peers {
		if peer.Status != Connected {
			err := peer.Connect()
			if err != nil {
				continue
			}
		}

		args := RequestVoteArgs{Term: term, CandidateID: s.addr}
		var reply RequestVoteReply
		res := requestVote(peer, &args, &reply)
		if res.Error != nil {
			total--
		}

		go func(peer *Client) {
			if res.Ok {
				s.muRaft.Lock()
				defer s.muRaft.Unlock()

				if reply.Term > s.term {
					s.term = reply.Term
					s.state = Follower
					s.votedFor = ""
					return
				}

				if s.state == Candidate && reply.VoteGranted && term == s.term {
					votes++
					needed := majority(total)
					if votes >= needed {
						s.becomeLeader()
					}
				}
			}
		}(peer)
	}
}

/**
* becomeLeader
**/
func (s *Raft) becomeLeader() {
	s.state = Leader
	s.leaderID = s.addr
	s.lastHeartbeat = timezone.Now()
	logs.Logf(packageName, "I am leader %s", s.addr)

	go s.heartbeatLoop()

	for _, fn := range s.server.onBecomeLeader {
		fn(s.server)
	}
}

/**
* heartbeatLoop
**/
func (s *Raft) heartbeatLoop() {
	if len(s.peers) == 0 {
		return
	}

	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	for range ticker.C {
		s.muRaft.Lock()
		state := s.state
		term := s.term
		s.muRaft.Unlock()
		if state != Leader {
			return
		}

		for _, peer := range s.peers {
			if peer.Addr == s.addr {
				continue
			}

			go func(peer *Client) {
				args := HeartbeatArgs{Term: term, LeaderID: s.addr}
				var reply HeartbeatReply
				res := heartbeat(peer, &args, &reply)
				if res.Ok {
					s.muRaft.Lock()
					defer s.muRaft.Unlock()

					if reply.Term > s.term {
						s.term = reply.Term
						s.state = Follower
						s.votedFor = ""
					}
				}
			}(peer)
		}
	}
}

/**
* requestVote
* @param to *Client, args *RequestVoteArgs, reply *RequestVoteReply
* @return error
**/
func (s *Raft) requestVote(args *RequestVoteArgs, reply *RequestVoteReply) error {
	s.muRaft.Lock()
	defer s.muRaft.Unlock()

	if args.Term < s.term {
		reply.Term = s.term
		reply.VoteGranted = false
		return nil
	}

	if args.Term > s.term {
		s.term = args.Term
		s.state = Follower
		s.votedFor = ""
	}

	if s.votedFor == "" || s.votedFor == args.CandidateID {
		s.votedFor = args.CandidateID
		reply.VoteGranted = true
		s.lastHeartbeat = timezone.Now()
	} else {
		reply.VoteGranted = false
	}

	reply.Term = s.term
	return nil
}

/**
* heartbeat
* @param args *HeartbeatArgs, reply *HeartbeatReply
* @return error
**/
func (s *Raft) heartbeat(args *HeartbeatArgs, reply *HeartbeatReply) error {
	changedLeader := false

	s.muRaft.Lock()
	if args.Term < s.term {
		reply.Term = s.term
		reply.Ok = false
		s.muRaft.Unlock()
		return nil
	}

	if args.Term > s.term {
		s.term = args.Term
		s.votedFor = ""
	}

	oldLeader := s.leaderID
	s.state = Follower
	s.leaderID = args.LeaderID
	s.lastHeartbeat = timezone.Now()

	if oldLeader != args.LeaderID {
		changedLeader = true
	}

	reply.Term = s.term
	reply.Ok = true
	s.muRaft.Unlock()

	if changedLeader {
		for _, fn := range s.server.onChangeLeader {
			fn(s.server)
		}
	}
	return nil
}

/**
* requestVote
* @param to *Client, require *RequestVoteArgs, response *RequestVoteReply
* @return *ResponseBool
**/
func requestVote(to *Client, require *RequestVoteArgs, response *RequestVoteReply) *ResponseBool {
	m, err := NewMessage(RequestVote, require)
	if err != nil {
		return &ResponseBool{
			Ok:    false,
			Error: err,
		}
	}

	msg, err := to.request(m)
	if err != nil {
		return &ResponseBool{
			Ok:    false,
			Error: err,
		}
	}

	err = msg.Get(&response)
	if err != nil {
		return &ResponseBool{
			Ok:    false,
			Error: err,
		}
	}

	return &ResponseBool{
		Ok:    true,
		Error: nil,
	}
}

/**
* heartbeat: Sends a heartbeat
* @param to *Client, require *HeartbeatArgs, response *HeartbeatReply
* @return error
**/
func heartbeat(to *Client, require *HeartbeatArgs, response *HeartbeatReply) *ResponseBool {
	m, err := NewMessage(Heartbeat, require)
	if err != nil {
		return &ResponseBool{
			Ok:    false,
			Error: err,
		}
	}

	msg, err := to.request(m)
	if err != nil {
		return &ResponseBool{
			Ok:    false,
			Error: err,
		}
	}

	err = msg.Get(&response)
	if err != nil {
		return &ResponseBool{
			Ok:    false,
			Error: err,
		}
	}

	return &ResponseBool{
		Ok:    true,
		Error: nil,
	}
}

/**
* newRaft
* @param srv *Server
* @return *Raft
**/
func newRaft(srv *Server) *Raft {
	this := &Raft{
		server:        srv,
		addr:          srv.addr,
		peers:         make([]*Client, 0),
		state:         Follower,
		term:          0,
		votedFor:      "",
		leaderID:      "",
		lastHeartbeat: time.Now(),
		turn:          0,
		muRaft:        sync.Mutex{},
		muTurn:        sync.Mutex{},
	}

	this.RequestVote = func(request *Message) *Response {
		var args RequestVoteArgs
		err := request.GetArgs(&args)
		if err != nil {
			return TcpError(err)
		}

		var response RequestVoteReply
		err = this.requestVote(&args, &response)
		if err != nil {
			return TcpError(err)
		}

		return TcpResponse(response)
	}

	this.Heartbeat = func(request *Message) *Response {
		var args HeartbeatArgs
		err := request.GetArgs(&args)
		if err != nil {
			return TcpError(err)
		}

		var response HeartbeatReply
		err = this.heartbeat(&args, &response)
		if err != nil {
			return TcpError(err)
		}

		return TcpResponse(response)
	}

	this.build()
	return this
}
