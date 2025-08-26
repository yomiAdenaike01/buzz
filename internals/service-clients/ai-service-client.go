package serviceClients

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/yomiAdenaike01/buzz-core/internals/config"
	"github.com/yomiAdenaike01/buzz-core/internals/store"
)

type AIServiceClient struct {
	config     config.AppConfig
	store      *store.Store
	httpClient *AIHttpClient
}

type SummariseResult struct {
	Result bool   `json:"result"`
	Text   string `json:"text"`
	Url    string `json:"url"`
}

type RawSummariseResult struct {
	Result string `json:"result"`
	Text   string `json:"text"`
}

func (a *AIServiceClient) SummarisePosts(posts []Post) ([]SummariseResult, []error) {
	var wg sync.WaitGroup
	summariesChan := make(chan SummariseResult)
	errChan := make(chan error)

	for _, post := range posts {
		wg.Add(1)
		go func(p Post) {
			defer wg.Done()
			body := map[string]any{
				"messages": []map[string]string{
					{
						"role": "system",
						"content": `You are a JSON-only generator. 
Summarize the AITA post, and output ONLY a valid JSON object with the exact structure:

{"text": "<summarised content>", "result": "<true|false>"}

Where:
- "text" is the summary (string).
- "result" is "true" if the user IS the asshole, or "false" if the user is NOT the asshole.
Do not include explanations, markdown, or any other text.`,
					},
					{
						"role":    "user",
						"content": fmt.Sprintf("Title: %s\nContent: %s", p.Title, p.Selftext),
					},
				},
			}
			data, err := json.Marshal(body)
			if err != nil {
				errChan <- err
				return
			}
			summary, err := a.httpClient.Summarise(data)

			if err != nil {
				errChan <- err
				return
			}
			if summary.Text == "" {
				return
			}
			summary.Url = fmt.Sprintf("https://reddit.com%s", p.Permalink)
			summariesChan <- summary
		}(post)
	}

	go func() {
		wg.Wait()
		close(summariesChan)
		close(errChan)
	}()
	summariesList := make([]SummariseResult, 0, len(posts))
	for summary := range summariesChan {
		summariesList = append(summariesList, summary)
	}

	errorsList := make([]error, 0, len(posts))
	for er := range errChan {
		errorsList = append(errorsList, er)
	}
	return summariesList, errorsList
}

type AIHttpClient struct {
	baseClient *http.Client
	config     config.AppConfig
}

type APIResponse struct {
	Result struct {
		Response string `json:"response"`
		Usage    struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	} `json:"result"`
	Success  bool     `json:"success"`
	Errors   []string `json:"errors"`
	Messages []string `json:"messages"`
}

func (a *AIHttpClient) Summarise(instruction []byte) (SummariseResult, error) {
	log.Printf("[Summarise] Summarising from instruction")
	var result SummariseResult
	url := &url.URL{
		Scheme: "https",
		Host:   "api.cloudflare.com",
		Path:   fmt.Sprintf("/client/v4/accounts/%s/ai/run/%s", a.config.CLOUD_FLARE_ACCOUNT_ID, a.config.CLOUD_FLARE_AI_MODEL),
	}
	request, err := http.NewRequest("POST", url.String(), bytes.NewReader(instruction))
	if err != nil {
		return result, fmt.Errorf("failed to build request=%w", err)
	}
	request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", a.config.CLOUD_FLARE_API_TOKEN))

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return result, fmt.Errorf("failed to send request err=%w", err)
	}
	defer response.Body.Close()
	var apiResp APIResponse
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return result, err
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		log.Printf("failed to unmarshal outer: err=%s", err.Error())
		return result, err
	}

	var summary RawSummariseResult
	if err := json.Unmarshal([]byte(apiResp.Result.Response), &summary); err != nil {
		log.Printf("failed to unmarshal inner: err=%s", err.Error())
		return result, err
	}
	result.Text = summary.Text
	if strings.EqualFold(summary.Result, "true") {
		result.Result = true
	}
	return result, nil
}

func newHttpClient(config config.AppConfig) *AIHttpClient {
	return &AIHttpClient{
		baseClient: &http.Client{},
		config:     config,
	}
}

func newAIServiceClient(config config.AppConfig, store *store.Store) *AIServiceClient {
	httpClient := newHttpClient(config)
	return &AIServiceClient{
		config:     config,
		store:      store,
		httpClient: httpClient,
	}
}
