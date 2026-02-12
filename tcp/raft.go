package tcp

import (
	"math/rand"
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

/**
* GetLeader
* @return string, error
**/
func (s *Server) GetLeader() (string, bool) {
	s.muCluster.Lock()
	inCluster := len(s.peers) > 1
	result := s.leaderID
	s.muCluster.Unlock()
	if !inCluster {
		return result, true
	}
	return result, result != "" && result == s.address
}

/**
* ElectionLoop
**/
func (s *Server) ElectionLoop() {
	if len(s.peers) == 0 {
		s.muCluster.Lock()
		s.becomeLeader()
		s.muCluster.Unlock()
		return
	}

	s.muCluster.Lock()
	s.state = Follower
	s.lastHeartbeat = timezone.Now()
	s.muCluster.Unlock()

	for {
		timeout := randomBetween(1500, 3000)
		time.Sleep(timeout)

		s.muCluster.Lock()
		elapsed := time.Since(s.lastHeartbeat)
		state := s.state
		s.muCluster.Unlock()

		if elapsed > heartbeatInterval && state != Leader {
			s.startElection()
		}
	}
}

/**
* startElection
**/
func (s *Server) startElection() {
	s.muCluster.Lock()
	s.state = Candidate
	s.term++
	term := s.term
	s.votedFor = s.address
	s.muCluster.Unlock()

	votes := 1
	total := len(s.peers)
	for _, peer := range s.peers {
		if peer.Status != Connected {
			err := peer.Connect()
			if err != nil {
				s.error(peer, err)
				continue
			}
		}

		go func(peer *Client) {
			args := RequestVoteArgs{Term: term, CandidateID: s.address}
			var reply RequestVoteReply
			res := s.requestVote(peer, &args, &reply)
			if res.Error != nil {
				total--
			}

			if res.Ok {
				s.muCluster.Lock()
				defer s.muCluster.Unlock()

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
func (s *Server) becomeLeader() {
	s.state = Leader
	s.leaderID = s.address
	s.lastHeartbeat = timezone.Now()
	logs.Logf(packageName, "I am leader %s", s.address)

	go s.heartbeatLoop()

	for _, fn := range s.onBecomeLeader {
		fn(s)
	}
}

/**
* heartbeatLoop
**/
func (s *Server) heartbeatLoop() {
	if len(s.peers) == 0 {
		return
	}

	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	for range ticker.C {
		s.muCluster.Lock()
		state := s.state
		term := s.term
		s.muCluster.Unlock()
		if state != Leader {
			return
		}

		for _, peer := range s.peers {
			if peer.Addr == s.address {
				continue
			}

			go func(peer *Client) {
				args := HeartbeatArgs{Term: term, LeaderID: s.address}
				var reply HeartbeatReply
				res := heartbeat(peer, &args, &reply)
				if res.Ok {
					s.muCluster.Lock()
					defer s.muCluster.Unlock()

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
func (s *Server) requestVote(to *Client, args *RequestVoteArgs, reply *RequestVoteReply) *ResponseBool {
	msg, err := s.Request(to, RequestVote, args, 10*time.Second)
	if err != nil {
		logs.Debugf("requestVote: %s | error: %s", msg.ToJson().ToString(), err.Error())
		return &ResponseBool{
			Ok:    false,
			Error: err,
		}
	}

	logs.Debug("requestVote:", msg.ToJson().ToString())
	// s.muCluster.Lock()
	// defer s.muCluster.Unlock()

	// if args.Term < s.term {
	// 	reply.Term = s.term
	// 	reply.VoteGranted = false
	// 	return nil
	// }

	// if args.Term > s.term {
	// 	s.term = args.Term
	// 	s.state = Follower
	// 	s.votedFor = ""
	// }

	// if s.votedFor == "" || s.votedFor == args.CandidateID {
	// 	s.votedFor = args.CandidateID
	// 	reply.VoteGranted = true
	// 	s.lastHeartbeat = timezone.Now()
	// } else {
	// 	reply.VoteGranted = false
	// }

	// reply.Term = s.term
	return &ResponseBool{
		Ok:    true,
		Error: nil,
	}
}

/**
* heartbeat
* @param args *HeartbeatArgs, reply *HeartbeatReply
* @return error
**/
func (s *Server) heartbeat(args *HeartbeatArgs, reply *HeartbeatReply) error {
	changedLeader := false

	s.muCluster.Lock()
	if args.Term < s.term {
		reply.Term = s.term
		reply.Ok = false
		s.muCluster.Unlock()
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
	s.muCluster.Unlock()

	if changedLeader {
		for _, fn := range s.onChangeLeader {
			fn(s)
		}
	}
	return nil
}

/**
* RequestVote: Requests a vote
* @param require *RequestVoteArgs, response *RequestVoteReply
* @return error
**/
// func (s *Server) RequestVote(require *RequestVoteArgs, response *RequestVoteReply) error {
// 	err := s.requestVote(require, response)
// 	return err
// }

/**
* heartbeat: Sends a heartbeat
* @param to *Client, require *HeartbeatArgs, response *HeartbeatReply
* @return error
**/
func heartbeat(to *Client, require *HeartbeatArgs, response *HeartbeatReply) *ResponseBool {
	// // var res HeartbeatReply
	// // err := jrpc.Call(to, "Node.Heartbeat", require, &res)
	// // if err != nil {
	// // 	return &ResponseBool{
	// // 		Ok:    false,
	// // 		Error: err,
	// // 	}
	// // }

	// *response = res
	return &ResponseBool{
		Ok:    true,
		Error: nil,
	}
}

/**
* Heartbeat: Sends a heartbeat
* @param require *HeartbeatArgs, response *HeartbeatReply
* @return error
**/
func (s *Server) Heartbeat(require *HeartbeatArgs, response *HeartbeatReply) error {
	err := s.heartbeat(require, response)
	return err
}

/**
* onChangeLeader
**/
func (s *Server) OnChangeLeader(fn func(*Server)) {
	s.onChangeLeader = append(s.onChangeLeader, fn)
}

/**
* OnBecomeLeader
**/
func (s *Server) OnBecomeLeader(fn func(*Server)) {
	s.onBecomeLeader = append(s.onBecomeLeader, fn)
}

/**
* OnBecomeFollower
**/
func (s *Server) OnBecomeFollower(fn func(*Server)) {
	s.onBecomeLeader = append(s.onBecomeLeader, fn)
}
