package notification

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"service-uptime-center/internal/app/util"
)

var (
	ErrMissingNtfyConfigProperty = fmt.Errorf("missing required property in ntfy config")
	ErrInvalidNtfyServer         = fmt.Errorf("invalid ntfy server")
)

type NtfyConfig struct {
	Server    string `toml:"server"`
	Topic     string `toml:"topic"`
	TokenFile string `toml:"token_file"`
	token     string
}

func (n *NtfyConfig) Validate() error {
	if strings.TrimSpace(n.Server) == "" {
		return fmt.Errorf("%w: Server", ErrMissingNtfyConfigProperty)
	}
	if strings.TrimSpace(n.Topic) == "" {
		return fmt.Errorf("%w: Topic", ErrMissingNtfyConfigProperty)
	}
	if strings.Contains(n.Server, " ") {
		return fmt.Errorf("%w: %s", ErrInvalidNtfyServer, n.Server)
	}
	if n.TokenFile != "" {
		token, err := util.ParsePasswordFile(n.TokenFile)
		if err != nil {
			return err
		}
		n.token = token
	}
	return nil
}

type ntfyNotifier struct {
	cfg    *NtfyConfig
	client *http.Client
}

func newNtfyNotifier(cfg *NtfyConfig) *ntfyNotifier {
	return &ntfyNotifier{
		cfg: cfg,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (n *ntfyNotifier) send(data SendData) error {
	server := strings.TrimRight(n.cfg.Server, "/")
	url := fmt.Sprintf("%s/%s", server, n.cfg.Topic)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBufferString(data.Body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "text/plain; charset=utf-8")
	if data.Title != "" {
		req.Header.Set("Title", data.Title)
	}
	if n.cfg.token != "" {
		req.Header.Set("Authorization", "Bearer "+n.cfg.token)
	}

	resp, err := n.client.Do(req)
	if err != nil {
		slog.Error("failed to send ntfy notification.", "server", n.cfg.Server, "topic", n.cfg.Topic)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		body, _ := io.ReadAll(resp.Body)
		slog.Error("ntfy notification failed.", "status", resp.StatusCode, "body", string(body))
		return fmt.Errorf("ntfy response status: %d", resp.StatusCode)
	}

	return nil
}
