package integration

import (
	"context"
	"net/url"
	"os"
	"testing"
	"time"

	"cdr.dev/slog"
	"cdr.dev/slog/sloggers/slogtest"
	"cdr.dev/slog/sloggers/slogtest/assert"

	"cdr.dev/coder-cli/coder-sdk"
)

func newClient(t *testing.T) *coder.Client {
	token := os.Getenv("CODER_TOKEN")
	if token == "" {
		slogtest.Fatal(t, `"CODER_TOKEN" env var is empty`)
	}
	raw := os.Getenv("CODER_URL")
	u, err := url.Parse(raw)
	if err != nil {
		slogtest.Fatal(t, `"CODER_URL" env var is invalid`, slog.Error(err))
	}

	return &coder.Client{
		BaseURL: u,
		Token:   token,
	}
}

func TestConfig(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	client := newClient(t)

	version, err := client.APIVersion(ctx)
	assert.Success(t, "get api version", err)
	slogtest.Info(t, "got api version", slog.F("version", version))

	authConfig, err := client.SiteConfigAuth(ctx)
	assert.Success(t, "auth config", err)
	slogtest.Info(t, "got site auth config", slog.F("config", authConfig))

	oauthConf, err := client.SiteConfigOAuth(ctx)
	assert.Success(t, "auth config", err)
	slogtest.Info(t, "got site oauth config", slog.F("config", oauthConf))

	putOAuth := &coder.ConfigOAuth{
		GitHub: coder.ConfigOAuthGitHub{
			BaseURL:      "github.com",
			ClientID:     "fake client id",
			ClientSecret: "fake secrets",
		},
	}

	err = client.PutSiteConfigOAuth(ctx, *putOAuth)
	assert.Success(t, "put site oauth", err)

	oauthConf, err = client.SiteConfigOAuth(ctx)
	assert.Success(t, "auth config", err)
	assert.Equal(t, "oauth was updated", putOAuth, oauthConf)
}
