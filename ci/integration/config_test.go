package integration

import (
	"context"
	"net/url"
	"testing"
	"time"

	"cdr.dev/slog"
	"cdr.dev/slog/sloggers/slogtest"
	"cdr.dev/slog/sloggers/slogtest/assert"
	"github.com/stretchr/testify/require"

	"cdr.dev/coder-cli/coder-sdk"
)

func TestConfig(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	creds := login(ctx, t)
	baseURL, err := url.Parse(creds.url)
	require.NoError(t, err, "error parsing baseURL")
	require.NotEmpty(t, creds.token, "session token is empty")
	client := &coder.Client{
		BaseURL: baseURL,
		Token:   creds.token,
	}

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
