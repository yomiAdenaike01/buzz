package serviceClients

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
)

type RedditServiceClient struct{}

type Listing struct {
	Data struct {
		Children []struct {
			Kind string `json:"kind"`
			Data Post   `json:"data"`
		} `json:"children"`
	} `json:"data"`
}

type Post struct {
	Title             string  `json:"title"`
	Selftext          string  `json:"selftext"`
	Permalink         string  `json:"permalink"`
	URL               string  `json:"url"`
	Author            string  `json:"author"`
	IsSelf            bool    `json:"is_self"`
	Stickied          bool    `json:"stickied"`
	Distinguished     string  `json:"distinguished"`
	LinkFlairText     string  `json:"link_flair_text"`
	RemovedByCategory *string `json:"removed_by_category"`
	IsSponsored       bool    `json:"is_sponsored"` // may not appear; default false
}

func isAITAStory(p Post) bool {
	if p.Stickied {
		return false
	}
	if p.Distinguished != "" {
		return false
	}
	if p.Author == "AutoModerator" {
		return false
	}
	if p.RemovedByCategory != nil && *p.RemovedByCategory != "" {
		return false
	}
	if p.IsSponsored {
		return false
	}
	if !p.IsSelf {
		return false
	}
	if len(strings.TrimSpace(p.Selftext)) < 120 {
		return false
	}
	return true
}

func (r *RedditServiceClient) GetPopularSubredditPosts(subreddit string, limit int) ([]Post, error) {
	log.Printf("[GetPopularSubredditPosts] Fetching numPosts=%d posts from subreddit=%s", limit, subreddit)
	url := fmt.Sprintf("https://www.reddit.com/r/%s/hot.json?limit=%d", subreddit, limit)
	req, err := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "linux:com.yomi.redditfetch:1.0.0 (by /u/yourname)")
	req.Header.Set("Accept", "application/json")
	if err != nil {
		return nil, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, errors.New("failed to fetch from reddit")
	}

	defer res.Body.Close()
	var listing Listing
	if err := json.NewDecoder(res.Body).Decode(&listing); err != nil {
		return nil, err
	}
	posts := make([]Post, 0, len(listing.Data.Children))
	for _, post := range listing.Data.Children {
		if isAITAStory(post.Data) {
			posts = append(posts, post.Data)
		}

	}
	return posts, nil
}

func newRedditServiceClient() *RedditServiceClient {
	return &RedditServiceClient{}
}
