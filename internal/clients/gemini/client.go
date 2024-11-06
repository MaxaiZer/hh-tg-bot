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

type Client struct {
	client      *genai.Client
	model       *genai.GenerativeModel
	rateLimiter *rate.Limiter
}

func NewClient(ctx context.Context, apiKey string) (*Client, error) {

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, err
	}

	model := client.GenerativeModel("gemini-1.5-flash")

	service := Client{
		client: client,
		model:  model,
	}

	return &service, nil
}

func (c *Client) SetRateLimit(maxRequestsPerSecond float32) {
	c.rateLimiter = rate.NewLimiter(rate.Limit(maxRequestsPerSecond), 1)
}

func (c *Client) GenerateResponse(ctx context.Context, text string) (string, error) {

	var resp string
	var err error

	_, _, _ = lo.AttemptWhileWithDelay(3, 2*time.Second, func(i int, _ time.Duration) (error, bool) {
		if i > 0 {
			log.Infof("gemini api returned 500 error, retrying...")
		}
		resp, err = c.waitAndGenerateResponse(ctx, text)
		return err, isInternalError(err)
	})

	return resp, err
}

func (c *Client) waitAndGenerateResponse(ctx context.Context, text string) (string, error) {

	if c.rateLimiter != nil {
		err := c.rateLimiter.Wait(ctx)
		if err != nil {
			return "", err
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
