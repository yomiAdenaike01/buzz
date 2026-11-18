package game

import (
	"sync"
)

type Vote struct {
	VoterID   string `json:"voter_id"`
	Candidate string `json:"candidate"`
}

type VotingRound struct {
	votes        []Vote
	done         chan struct{}
	once         sync.Once
	sw           sync.RWMutex
	expcetedSize int
}

type VotingResult struct {
	WinnerId      string         `json:"winner"`
	VotesByPlayer map[string]int `json:"votes"`
	TiedPlayers   []string       `json:"tied"`
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

func (v *VotingRound) Result() (result VotingResult) {
	v.sw.RLock()
	defer v.sw.RUnlock()
	result.VotesByPlayer = make(map[string]int, v.expcetedSize)

	maxVotes := 0
	for _, vote := range v.votes {
		currentVoteCount := result.VotesByPlayer[vote.Candidate]
		newVoteCount := currentVoteCount + 1
		result.VotesByPlayer[vote.Candidate] = newVoteCount

		if newVoteCount > maxVotes {
			maxVotes = newVoteCount
			result.WinnerId = vote.Candidate
			result.TiedPlayers = nil
			continue
		}

		if newVoteCount == maxVotes {
			result.TiedPlayers = append(result.TiedPlayers, vote.Candidate)
		}

	}
	if len(result.TiedPlayers) > 0 {
		result.WinnerId = ""
	}
	return
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
		sw:           sync.RWMutex{},
	}
}
