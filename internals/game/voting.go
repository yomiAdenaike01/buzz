package game

import "sync"

type Vote struct {
	VoterID   string `json:"voter_id"`
	Candidate string `json:"candidate"`
	Tie       bool   `json:"-"`
}

type VotingRound struct {
	votes        []Vote
	done         chan struct{}
	once         sync.Once
	sw           sync.Mutex
	expcetedSize int
}

type VotingResult struct {
	WinnerId      string         `json:"winner"`
	VotesByPlayer map[string]int `json:"votes"`
}

func (v *VotingRound) Done() <-chan struct{} {
	return v.done
}

func (v *VotingRound) Complete() {
	v.sw.Lock()
	v.once.Do(func() {
		close(v.done)
	})
	v.sw.Unlock()
}

func (v *VotingRound) Result() VotingResult {
	result := VotingResult{
		WinnerId:      "",
		VotesByPlayer: make(map[string]int, v.expcetedSize),
	}
	v.sw.Lock()
	for _, vote := range v.votes {
		candidate := vote.Candidate
		_, ok := result.VotesByPlayer[candidate]
		if !ok {
			result.VotesByPlayer[candidate] = 0
		}
		result.VotesByPlayer[candidate] = result.VotesByPlayer[candidate] + 1
	}
	v.sw.Unlock()
	max := 0
	for k, v := range result.VotesByPlayer {
		if v > max {
			result.WinnerId = k
			max = v
		}
		if max == v && v > 0 {

		}
	}
	return result
}

func (v *VotingRound) SubmitVote(vote Vote) {
	v.votes = append(v.votes, vote)
	if len(v.votes) == cap(v.votes) || len(v.votes) >= v.expcetedSize {
		v.Complete()
	}
}

func NewVotingRound(questionText string, numPlayers int) *VotingRound {
	return &VotingRound{
		votes:        make([]Vote, 0, numPlayers),
		done:         make(chan struct{}),
		expcetedSize: numPlayers,
		once:         sync.Once{},
		sw:           sync.Mutex{},
	}
}
