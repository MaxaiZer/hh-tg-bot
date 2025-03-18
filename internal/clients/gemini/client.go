package gemini

import (
	"context"
	"fmt"
	"github.com/google/generative-ai-go/genai"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
	"google.golang.org/api/option"
	"strings"
	"time"
)

type Model string

const (
	//Model15Flash is fastest multimodal model with great performance for diverse, repetitive tasks
	Model15Flash Model = "gemini-1.5-flash"
	//Model15Flash8b is the smallest model for lower intelligence use cases
	Model15Flash8b Model = "gemini-1.5-flash-8b"
	//Model15Pro is next-generation model with a breakthrough 2 million context window
	Model15Pro Model = "gemini-1.5-pro"
	//Model10Pro is first-generation model offering only text and image reasoning
	Model10Pro Model = "gemini-1.0-pro"
)

type Client struct {
	client            *genai.Client
	model             *genai.GenerativeModel
	minuteRateLimiter *rate.Limiter
	dayRateLimiter    *rate.Limiter
}

func NewClient(ctx context.Context, apiKey string, model Model) (*Client, error) {

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, err
	}

	genModel := client.GenerativeModel(string(model))

	service := Client{
		client: client,
		model:  genModel,
	}

	return &service, nil
}

func (c *Client) SetMinuteRateLimit(maxRequestsPerMinute float32) {
	c.minuteRateLimiter = rate.NewLimiter(rate.Limit(maxRequestsPerMinute/60), 1)
}

func (c *Client) SetDayRateLimit(maxRequestsPerDay float32) {
	c.dayRateLimiter = rate.NewLimiter(rate.Limit(maxRequestsPerDay/86400), int(maxRequestsPerDay))
}

func (c *Client) GenerateResponse(ctx context.Context, text string) (string, error) {

	var resp string
	var err error

	_, _, _ = lo.AttemptWhileWithDelay(3, 2*time.Second, func(i int, _ time.Duration) (error, bool) {
		if i > 0 {
			log.Warn("gemini api returned 500 error, retrying...")
		}
		resp, err = c.waitAndGenerateResponse(ctx, text)
		return err, isInternalError(err)
	})

	return resp, err
}

func (c *Client) waitAndGenerateResponse(ctx context.Context, text string) (string, error) {

	limiters := []*rate.Limiter{c.minuteRateLimiter, c.dayRateLimiter}
	for _, limiter := range limiters {
		if limiter != nil {
			err := limiter.Wait(ctx)
			if err != nil {
				return "", err
			}
		}
	}

	resp, err := c.tryGenerateResponse(ctx, text)
	if err != nil {
		return "", err
	}
	return resp, nil
}

func (c *Client) tryGenerateResponse(ctx context.Context, text string) (string, error) {

	response, err := c.model.GenerateContent(ctx, genai.Text(text))
	if err != nil {
		return "", err
	}

	part := response.Candidates[0].Content.Parts[0]

	if textPart, ok := part.(genai.Text); ok {
		return string(textPart), nil
	}

	return "", fmt.Errorf("response part is not text")
}

func isInternalError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "Error 500")
}
