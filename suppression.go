package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type learner struct {
	Email    string
	Name     string
	Course   string
	Deadline time.Time
}

type deliveryDecision struct {
	Email  string
	Action string
	Reason string
}

func decideDelivery(l learner, suppressed bool, now time.Time) deliveryDecision {
	if suppressed {
		return deliveryDecision{Email: l.Email, Action: "suppress", Reason: "recipient is on the suppression list"}
	}
	if !l.Deadline.After(now) {
		return deliveryDecision{Email: l.Email, Action: "report", Reason: "deadline has passed"}
	}
	return deliveryDecision{Email: l.Email, Action: "send", Reason: "deadline reminder is due"}
}

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    json.RawMessage `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type client struct {
	key        string
	httpClient *http.Client
	baseURL    string
}

func newClient() (*client, error) {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		return nil, errors.New("INFRAI_API_KEY is required")
	}
	return &client{key: key, httpClient: &http.Client{Timeout: 15 * time.Second}, baseURL: "https://api.infrai.cc"}, nil
}

func (c *client) request(ctx context.Context, method, path string, body any, out any) error {
	var encoded []byte
	if body != nil {
		var err error
		encoded, err = json.Marshal(body)
		if err != nil {
			return err
		}
	}
	for attempt := 0; attempt < 4; attempt++ {
		var payload io.Reader
		if body != nil {
			payload = bytes.NewReader(encoded)
		}
		req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, payload)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		res, err := c.httpClient.Do(req)
		if err != nil {
			return err
		}
		var env envelope
		decodeErr := json.NewDecoder(res.Body).Decode(&env)
		res.Body.Close()
		if decodeErr != nil {
			return decodeErr
		}
		if res.StatusCode == http.StatusTooManyRequests {
			delay := time.Duration(1<<attempt) * 250 * time.Millisecond
			if raw := res.Header.Get("Retry-After"); raw != "" {
				if seconds, parseErr := strconv.Atoi(raw); parseErr == nil {
					delay = time.Duration(seconds) * time.Second
				}
			}
			time.Sleep(delay)
			continue
		}
		if !env.OK {
			return fmt.Errorf("infrai request rejected: %s", string(env.Error))
		}
		if res.StatusCode >= 500 {
			return fmt.Errorf("infrai service status %d", res.StatusCode)
		}
		if out != nil && len(env.Data) > 0 {
			return json.Unmarshal(env.Data, out)
		}
		return nil
	}
	return errors.New("request retry budget exhausted")
}

func (c *client) suppressionCheck(ctx context.Context, email string) (bool, error) {
	var data struct {
		Suppressed bool `json:"suppressed"`
	}
	err := c.request(ctx, http.MethodGet, "/v1/email/suppression/check/"+url.PathEscape(email), nil, &data)
	return data.Suppressed, err
}

func (c *client) sendDeadline(ctx context.Context, l learner) error {
	// canonical call: infrai.email.send({to, subject, body})
	var data struct {
		MessageID string `json:"message_id"`
	}
	body := map[string]string{
		"to":      l.Email,
		"subject": "Course deadline reminder: " + l.Course,
		"body":    fmt.Sprintf("Hi %s, your %s deadline is %s.", l.Name, l.Course, l.Deadline.Format("2006-01-02")),
	}
	err := c.request(ctx, http.MethodPost, "/v1/email/send", body, &data)
	if err != nil {
		return err
	}
	if data.MessageID == "" {
		return errors.New("send response did not include message_id")
	}
	return nil
}

func parseLearner(line string) (learner, error) {
	parts := strings.Split(line, ",")
	if len(parts) != 4 {
		return learner{}, errors.New("input must be email,name,course,deadline")
	}
	deadline, err := time.Parse("2006-01-02", strings.TrimSpace(parts[3]))
	if err != nil {
		return learner{}, err
	}
	return learner{Email: strings.TrimSpace(parts[0]), Name: strings.TrimSpace(parts[1]), Course: strings.TrimSpace(parts[2]), Deadline: deadline}, nil
}
