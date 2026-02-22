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
	mu            sync.Mutex             `json:"-"`
	muTurn        sync.Mutex             `json:"-"`
}

/**
* build
* @return map[string]HandlerFunc
**/
func (s *Raft) build() map[string]HandlerFunc {
	s.registry = map[string]HandlerFunc{}
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
	s.mu.Lock()
	s.peers = append(s.peers, node)
	s.mu.Unlock()
}

/**
* removeNode
* @param addr string
**/
func (s *Raft) removeNode(addr string) {
	s.mu.Lock()
	defer s.mu.Unlock()

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
	// s.mu.Lock()
	leader = s.leaderID
	state := s.state
	// s.mu.Unlock()
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

	// s.mu.Lock()
	result := s.peers[idx]
	// s.mu.Unlock()

	return result, false
}

/**
* nextTurn
* @return *Client
**/
func (s *Raft) nextTurn() *Client {
	// s.muTurn.Lock()
	result := s.peers[s.turn]
	s.turn++
	// s.muTurn.Unlock()

	return result
}

/**
* electionLoop
**/
func (s *Raft) electionLoop() {
	if len(s.peers) == 0 {
		s.mu.Lock()
		s.becomeLeader()
		s.mu.Unlock()
		return
	}

	s.mu.Lock()
	s.state = Follower
	s.lastHeartbeat = timezone.Now()
	s.mu.Unlock()

	for {
		timeout := randomBetween(1500, 3000)
		time.Sleep(timeout)

		s.mu.Lock()
		elapsed := time.Since(s.lastHeartbeat)
		state := s.state
		s.mu.Unlock()

		if elapsed > heartbeatInterval && state != Leader {
			s.startElection()
		}
	}
}

/**
* startElection
**/
func (s *Raft) startElection() {
	// s.mu.Lock()
	s.state = Candidate
	s.term++
	term := s.term
	s.votedFor = s.addr
	// s.mu.Unlock()

	votes := 1
	total := len(s.peers)

	defer func() {
		logs.Debugf("startElection:%s total:%d", s.addr, total)
	}()

	for _, peer := range s.peers {
		if peer.Status != Connected {
			err := peer.Connect()
			if err != nil {
				total--
				continue
			}
		}

		args := RequestVoteArgs{Term: term, CandidateID: s.addr}
		var reply RequestVoteReply
		res := requestVote(peer, &args, &reply)
		if res.Error != nil {
			total--
			continue
		}

		go func(peer *Client) {
			if res.Ok {
				// s.mu.Lock()
				// defer s.mu.Unlock()

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

	// go s.heartbeatLoop()

	for _, fn := range s.server.onBecomeLeader {
		fn(s.server)
	}
}

/**
* heartbeatLoop
**/
func (s *Raft) heartbeatLoop() {
	logs.Debug("heartbeatLoop")

	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	for range ticker.C {
		// s.mu.Lock()
		state := s.state
		term := s.term
		// s.mu.Unlock()
		if state != Leader {
			return
		}

		for _, peer := range s.peers {
			if peer.Addr == s.addr {
				continue
			}

			if peer.Status != Connected {
				continue
			}

			go func(peer *Client) {
				args := HeartbeatArgs{Term: term, LeaderID: s.addr}
				var reply HeartbeatReply
				res := heartbeat(peer, &args, &reply)
				if res.Ok {
					// s.mu.Lock()
					// defer s.mu.Unlock()

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
	// s.mu.Lock()
	// defer s.mu.Unlock()

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

	// s.mu.Lock()
	if args.Term < s.term {
		reply.Term = s.term
		reply.Ok = false
		// s.mu.Unlock()
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
	// s.mu.Unlock()

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
		mu:            sync.Mutex{},
		muTurn:        sync.Mutex{},
	}
	this.build()
	return this
}
