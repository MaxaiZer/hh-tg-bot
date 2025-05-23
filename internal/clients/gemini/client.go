package gemini

import (
	"context"
	"fmt"
	"github.com/google/generative-ai-go/genai"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"strings"
	"time"
)

type Client struct {
	client            *genai.Client
	model             *genai.GenerativeModel
	minuteRateLimiter *rate.Limiter
	dayRateLimiter    *rate.Limiter
}

func NewClient(ctx context.Context, apiKey string, model string) (*Client, error) {

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, err
	}

	genModel := client.GenerativeModel(model)

	wrapper := Client{
		client: client,
		model:  genModel,
	}

	exists, err := wrapper.doesModelExist(ctx, model)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("model does not exist: %q", model)
	}

	return &wrapper, nil
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

func (c *Client) doesModelExist(ctx context.Context, name string) (bool, error) {

	iter := c.client.ListModels(ctx)
	for {
		model, err := iter.Next()
		if err == iterator.Done {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		if strings.TrimPrefix(model.Name, "models/") == name {
			return true, nil
		}
	}
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
