package tcp

import (
	"context"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cgalvisleon/et/color"
	"github.com/cgalvisleon/et/logs"
	mg "github.com/cgalvisleon/et/msg"
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
	ctx           context.Context    `json:"-"`
	cancel        context.CancelFunc `json:"-"`
	node          *Node              `json:"-"`
	addr          string             `json:"-"`
	state         Mode               `json:"-"`
	term          int                `json:"-"`
	votedFor      string             `json:"-"`
	leaderID      string             `json:"-"`
	lastHeartbeat time.Time          `json:"-"`
	mu            sync.Mutex         `json:"-"`
}

func newRaft(node *Node) *Raft {
	ctx, cancel := context.WithCancel(node.ctx)

	return &Raft{
		node:          node,
		addr:          node.addr,
		state:         Follower,
		term:          0,
		votedFor:      "",
		leaderID:      "",
		lastHeartbeat: time.Now(),
		ctx:           ctx,
		cancel:        cancel,
	}
}

/**
* getPeers
* @return []*Client
**/
func (s *Raft) getPeers() []*Client {
	return s.node.GetPeers()
}

/**
* LeaderID
* @return string, bool
**/
func (s *Raft) LeaderID() (leader string, imLeader bool) {
	s.mu.Lock()
	leader = s.leaderID
	state := s.state
	s.mu.Unlock()
	imLeader = state == Leader
	return
}

/**
* electionLoop
**/
func (s *Raft) electionLoop() {
	s.mu.Lock()
	s.lastHeartbeat = timezone.Now()
	s.mu.Unlock()

	for {
		timeout := randomBetween(1500, 3000)

		select {
		case <-time.After(timeout):
		case <-s.ctx.Done():
			return
		}

		s.mu.Lock()
		if s.state == Leader {
			s.mu.Unlock()
			continue
		}

		if time.Since(s.lastHeartbeat) < timeout {
			s.mu.Unlock()
			continue
		}

		s.mu.Unlock()
		s.startElection()
	}
}

/**
* startElection
**/
func (s *Raft) startElection() {
	s.mu.Lock()
	if s.state == Leader {
		s.mu.Unlock()
		return
	}

	s.state = Candidate
	s.term++
	term := s.term
	s.votedFor = s.addr
	s.leaderID = ""
	s.lastHeartbeat = timezone.Now()
	s.mu.Unlock()

	peers := s.getPeers()
	var votes atomic.Int32
	votes.Store(1)

	for _, peer := range peers {
		go func(peer *Client) {
			if peer.Status != Connected {
				err := peer.Connect()
				if err != nil {
					return
				}
			}

			args := RequestVoteArgs{
				Term:        term,
				CandidateID: s.addr,
			}

			var reply RequestVoteReply
			res := requestVote(peer, &args, &reply)
			if !res.Ok {
				return
			}

			s.mu.Lock()
			defer s.mu.Unlock()

			if reply.Term > s.term {
				s.term = reply.Term
				s.state = Follower
				s.votedFor = ""
				return
			}

			if s.state == Candidate && term == s.term && reply.VoteGranted {
				newVotes := votes.Add(1)

				needed := majority(int(s.node.total.Load()))
				if int(newVotes) >= needed && s.state == Candidate {
					s.becomeLeader()
				}
			}
		}(peer)
	}
}

/**
* becomeLeader
**/
func (s *Raft) becomeLeader() {
	if s.state == Leader {
		return
	}

	s.state = Leader
	s.leaderID = s.addr
	s.lastHeartbeat = time.Now()

	logs.Logf(packageName, color.Yellow(mg.MSG_TCP_BECAME_LEADER), s.addr, s.term)

	go s.heartbeatLoop()

	go func() {
		for _, fn := range s.node.onBecomeLeader {
			fn(s.node)
		}
	}()
}

/**
* heartbeatLoop
**/
func (s *Raft) heartbeatLoop() {
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
		case <-s.ctx.Done():
			return
		}

		s.mu.Lock()
		if s.state != Leader {
			s.mu.Unlock()
			continue
		}
		term := s.term
		s.mu.Unlock()

		peers := s.getPeers()

		for _, peer := range peers {
			if peer.Status != Connected {
				continue
			}

			go func(peer *Client) {
				args := HeartbeatArgs{
					Term:     term,
					LeaderID: s.addr,
				}

				var reply HeartbeatReply
				res := heartbeat(peer, &args, &reply)
				if !res.Ok {
					return
				}

				s.mu.Lock()
				defer s.mu.Unlock()

				if reply.Term > s.term {
					s.term = reply.Term
					s.state = Follower
					s.votedFor = ""
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
func (s *Raft) requestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if args.Term < s.term {
		reply.Term = s.term
		reply.VoteGranted = false
		return
	}

	if args.Term > s.term {
		s.term = args.Term
		s.state = Follower
		s.votedFor = ""
	}

	reply.Term = s.term

	if s.votedFor == "" || s.votedFor == args.CandidateID {
		s.votedFor = args.CandidateID
		reply.VoteGranted = true
		s.lastHeartbeat = timezone.Now()
	} else {
		reply.VoteGranted = false
	}
}

/**
* heartbeat
* @param args *HeartbeatArgs, reply *HeartbeatReply
**/
func (s *Raft) heartbeat(args *HeartbeatArgs, reply *HeartbeatReply) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if args.Term < s.term {
		reply.Term = s.term
		reply.Ok = false
		return
	}

	if args.Term > s.term {
		s.term = args.Term
		s.votedFor = ""
	}

	oldLeader := s.leaderID

	s.state = Follower
	s.leaderID = args.LeaderID
	s.lastHeartbeat = timezone.Now()

	reply.Term = s.term
	reply.Ok = true

	if oldLeader != args.LeaderID {
		logs.Logf(packageName, mg.MSG_TCP_CHANGED_LEADER, s.term, args.LeaderID)
		go func() {
			for _, fn := range s.node.onChangeLeader {
				fn(s.node)
			}
		}()
	}
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
* heartbeat:
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
