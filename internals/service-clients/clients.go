package serviceClients

import (
	"log"
	"strconv"

	"github.com/google/uuid"
	"github.com/yomiAdenaike01/buzz-core/internals/config"
	"github.com/yomiAdenaike01/buzz-core/internals/store"
	"github.com/yomiAdenaike01/buzz-core/internals/utils"
)

type ServiceClients struct {
	Reddit       *RedditServiceClient
	Notification *NotificationClient
	Ai           *AIServiceClient
}

func (s *ServiceClients) BuildChallengesFromPosts(store *store.Store) {
	log.Println("[BuildChallengesFromPosts]: Loading challenges from file")
	loaded := store.LoadRemoteChallengesFromFile()
	if loaded {
		return
	}
	log.Println("[BuildChallengesFromPosts] Starting to build challenges from reddit")
	posts, err := s.Reddit.GetPopularSubredditPosts("AmItheAsshole", 5)
	if err != nil {
		log.Printf("[BuildChallengesFromPosts] Failed to load challenges err=%s", err.Error())
		return
	}

	summaries, errors := s.Ai.SummarisePosts(posts)
	for _, e := range errors {
		log.Printf("[BuildChallengesFromPosts] Failed to summarise err=%s", e.Error())
	}
	if len(errors) > 0 {
		return
	}
	questions := make([]config.Question, 0, len(summaries))
	for _, summary := range summaries {
		questions = append(questions, config.Question{
			Id:       uuid.NewString(),
			Text:     summary.Text,
			Category: "AITA",
			Metadata: utils.MustJSON(map[string]string{
				"Answer": strconv.FormatBool(summary.Result),
				"Url":    summary.Url,
			}),
		})
	}

	store.SaveFetchedChallenges(questions)
}

func New(config config.AppConfig, store *store.Store) *ServiceClients {
	clients := &ServiceClients{
		Reddit:       newRedditServiceClient(),
		Ai:           newAIServiceClient(config, store),
		Notification: newNotificationClient(),
	}
	return clients
}
