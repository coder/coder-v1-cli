package cmd

import (
	"bytes"
	"context"
	"encoding/base64"
	"io/fs"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"testing"

	"cdr.dev/slog/sloggers/slogtest/assert"
	"github.com/manifoldco/promptui"
	"github.com/spf13/afero"
)

const (
	fakeExePath    = "/coder"
	fakeCoderURL   = "https://my.cdr.dev"
	fakeNewVersion = "1.23.4-rc.5+678-gabcdef-12345678"
	fakeOldVersion = "1.22.4-rc.5+678-gabcdef-12345678"
	fakeReleaseURL = "https://github.com/cdr/coder-cli/releases/download/v1.23.4/coder-cli-linux-amd64.tar.gz"
)

func Test_updater_run_noop(t *testing.T) {
	fakefs := afero.NewMemMapFs()
	httpClient := &fakeGetter{
		GetF: func(url string) (*http.Response, error) {
			switch url {
			case fakeCoderURL + "/api":
				return fakeResponse([]byte{}, 401, "coder-version: "+fakeNewVersion), nil
			default:
				t.Errorf("unhandled url: %s", url)
				t.FailNow()
				return nil, nil // this will never happen
			}
		},
	}
	ctx := context.Background()
	u := &updater{
		confirmF:       nil, // should not be required
		executablePath: fakeExePath,
		fs:             fakefs,
		httpClient:     httpClient,
		versionF:       func() string { return fakeNewVersion },
	}

	// write fake executable
	fakeFile(fakefs, fakeExePath, 0755, fakeNewVersion)
	err := u.Run(ctx, false, fakeCoderURL)
	assertFileContent(t, fakefs, fakeExePath, fakeNewVersion)
	assert.Success(t, "update coder - noop", err)
}

func Test_updater_run_changed(t *testing.T) {
	fakefs := afero.NewMemMapFs()
	httpClient := &fakeGetter{
		GetF: func(url string) (*http.Response, error) {
			switch url {
			case fakeCoderURL + "/api":
				return fakeResponse([]byte{}, 401, "coder-version: "+fakeNewVersion), nil
			case fakeReleaseURL:
				return fakeResponse(fakeValidTgzBytes, 200), nil
			default:
				t.Errorf("unhandled url: %s", url)
				t.FailNow()
				return nil, nil // this will never happen
			}
		},
	}
	ctx := context.Background()
	u := &updater{
		confirmF:       fakeConfirmYes,
		executablePath: fakeExePath,
		fs:             fakefs,
		httpClient:     httpClient,
		versionF:       func() string { return fakeOldVersion },
	}

	// write fake executable
	fakeFile(fakefs, fakeExePath, 0644, fakeOldVersion)
	err := u.Run(ctx, false, fakeCoderURL)
	assertFileContent(t, fakefs, fakeExePath, fakeNewVersion)
	assert.Success(t, "update coder - new version", err)
}

func Test_updater_run_changed_force(t *testing.T) {
	fakefs := afero.NewMemMapFs()
	fakeCoderURL := "https://my.cdr.dev"
	httpClient := &fakeGetter{
		GetF: func(url string) (*http.Response, error) {
			switch url {
			case fakeCoderURL + "/api":
				return fakeResponse([]byte{}, 401, "coder-version: "+fakeNewVersion), nil
			case fakeReleaseURL:
				return fakeResponse(fakeValidTgzBytes, 200), nil
			default:
				t.Errorf("unhandled url: %s", url)
				t.FailNow()
				return nil, nil // this will never happen
			}
		},
	}
	ctx := context.Background()
	u := &updater{
		confirmF:       nil, // should not be required
		executablePath: fakeExePath,
		fs:             fakefs,
		httpClient:     httpClient,
		versionF:       func() string { return fakeOldVersion },
	}

	// write fake executable
	fakeFile(fakefs, fakeExePath, 0644, fakeOldVersion)
	err := u.Run(ctx, true, fakeCoderURL)
	assertFileContent(t, fakefs, fakeExePath, fakeNewVersion)
	assert.Success(t, "update coder - new version", err)
}

func Test_updater_run_notconfirmed(t *testing.T) {
	fakefs := afero.NewMemMapFs()
	fakeCoderURL := "https://my.cdr.dev"
	httpClient := &fakeGetter{
		GetF: func(url string) (*http.Response, error) {
			switch url {
			case fakeCoderURL + "/api":
				return fakeResponse([]byte{}, 401, "coder-version: "+fakeNewVersion), nil
			default:
				t.Errorf("unhandled url: %s", url)
				t.FailNow()
				return nil, nil // this will never happen
			}
		},
	}
	ctx := context.Background()
	u := &updater{
		confirmF:       fakeConfirmNo,
		executablePath: fakeExePath,
		fs:             fakefs,
		httpClient:     httpClient,
		versionF:       func() string { return fakeOldVersion },
	}

	// write fake executable
	fakeFile(fakefs, fakeExePath, 0644, fakeOldVersion)
	err := u.Run(ctx, false, fakeCoderURL)
	assertFileContent(t, fakefs, fakeExePath, fakeOldVersion)
	assert.ErrorContains(t, "update coder - new version", err, "failed to confirm update")
}

type fakeGetter struct {
	GetF func(url string) (*http.Response, error)
}

func (f *fakeGetter) Get(url string) (*http.Response, error) {
	return f.GetF(url)
}

func fakeConfirmYes(_ string) (string, error) {
	return "y", nil
}

func fakeConfirmNo(_ string) (string, error) {
	return "", promptui.ErrAbort
}

func fakeResponse(body []byte, code int, headers ...string) *http.Response {
	resp := &http.Response{}
	resp.Body = ioutil.NopCloser(bytes.NewReader(body))
	resp.StatusCode = code
	resp.Header = http.Header{}

	for _, e := range headers {
		parts := strings.Split(e, ":")
		k := strings.ToLower(strings.TrimSpace(parts[0]))
		v := strings.ToLower(strings.TrimSpace(strings.Join(parts[1:], ":")))
		resp.Header.Set(k, v)
	}

	return resp
}

//nolint:unparam
func fakeFile(fs afero.Fs, name string, perm fs.FileMode, content string) {
	f, err := fs.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	_, err = f.Write([]byte(content))
	if err != nil {
		panic(err)
	}
}

//nolint:unparam
func assertFileContent(t *testing.T, fs afero.Fs, name string, content string) {
	f, err := fs.OpenFile(name, os.O_RDONLY, 0)
	assert.Success(t, "open file "+name, err)
	defer f.Close()

	b, err := ioutil.ReadAll(f)
	assert.Success(t, "read file "+name, err)

	assert.Equal(t, "assert content equal", content, string(b))
}

// this is a valid tgz file containing a single file named 'coder' with permissions 0751
// containing the string "1.23.4-rc.5+678-gabcdef-12345678".
var fakeValidTgzBytes, _ = base64.StdEncoding.DecodeString(`H4sIAAAAAAAAA+3QsQ4CIRCEYR6F3oC7wIqvc3KnpQnq+3tGCwsTK3LN/zWTTDWZuG/XeeluJFlV
s1dqNfnOtyJOi4qllHOuTlSTqPMydNXH43afuvfu3w3jb9qExpRjCb1F2x3qMVymU5uXc9CUi63F
1vsAAAAAAAAAAAAAAAAAAL89AYuL424AKAAA`)
