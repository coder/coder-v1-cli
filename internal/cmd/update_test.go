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

func Test_updater_run(t *testing.T) {
	t.Parallel()

	type params struct {
		ConfirmF       func(string) (string, error)
		Ctx            context.Context
		ExecutablePath string
		Fakefs         afero.Fs
		HttpClient     *fakeGetter
		VersionF       func() string
	}

	fromParams := func(p *params) *updater {
		return &updater{
			confirmF:       p.ConfirmF,
			executablePath: p.ExecutablePath,
			fs:             p.Fakefs,
			httpClient:     p.HttpClient,
			versionF:       p.VersionF,
		}
	}

	run := func(t *testing.T, name string, fn func(t *testing.T, p *params)) {
		t.Logf("running %s", name)
		ctx := context.Background()
		fakefs := afero.NewMemMapFs()
		params := &params{
			// This must be overridden inside run()
			ConfirmF: func(string) (string, error) {
				t.Errorf("unhandled ConfirmF")
				t.FailNow()
				return "", nil
			},
			Ctx:            ctx,
			ExecutablePath: fakeExePath,
			Fakefs:         fakefs,
			HttpClient:     newFakeGetter(t),
			// This must be overridden inside run()
			VersionF: func() string {
				t.Errorf("unhandled VersionF")
				t.FailNow()
				return ""
			},
		}

		fn(t, params)
	}

	run(t, "update coder - noop", func(t *testing.T, p *params) {
		fakeFile(p.Fakefs, fakeExePath, 0755, fakeNewVersion)
		p.HttpClient.M[fakeCoderURL+"/api"] = newFakeGetterResponse([]byte{}, 401, variadicS("coder-version: "+fakeNewVersion), nil)
		p.VersionF = func() string { return fakeNewVersion }
		u := fromParams(p)
		err := u.Run(p.Ctx, false, fakeCoderURL)
		assert.Success(t, "update coder - noop", err)
		assertFileContent(t, p.Fakefs, fakeExePath, fakeNewVersion)
	})

	run(t, "update coder - old to new", func(t *testing.T, p *params) {
		fakeFile(p.Fakefs, fakeExePath, 0755, fakeOldVersion)
		p.HttpClient.M[fakeCoderURL+"/api"] = newFakeGetterResponse([]byte{}, 401, variadicS("coder-version: "+fakeNewVersion), nil)
		p.HttpClient.M[fakeReleaseURL] = newFakeGetterResponse(fakeValidTgzBytes, 200, variadicS(), nil)
		p.VersionF = func() string { return fakeOldVersion }
		p.ConfirmF = func(string) (string, error) { return "", nil }
		u := fromParams(p)
		assertFileContent(t, p.Fakefs, fakeExePath, fakeOldVersion)
		err := u.Run(p.Ctx, false, fakeCoderURL)
		assert.Success(t, "update coder - old to new", err)
		assertFileContent(t, p.Fakefs, fakeExePath, fakeNewVersion)
	})

	run(t, "update coder - old to new forced", func(t *testing.T, p *params) {
		fakeFile(p.Fakefs, fakeExePath, 0755, fakeOldVersion)
		p.HttpClient.M[fakeCoderURL+"/api"] = newFakeGetterResponse([]byte{}, 401, variadicS("coder-version: "+fakeNewVersion), nil)
		p.HttpClient.M[fakeReleaseURL] = newFakeGetterResponse(fakeValidTgzBytes, 200, variadicS(), nil)
		p.VersionF = func() string { return fakeOldVersion }
		u := fromParams(p)
		assertFileContent(t, p.Fakefs, fakeExePath, fakeOldVersion)
		err := u.Run(p.Ctx, true, fakeCoderURL)
		assert.Success(t, "update coder - old to new", err)
		assertFileContent(t, p.Fakefs, fakeExePath, fakeNewVersion)
	})

	run(t, "update coder - user cancelled", func(t *testing.T, p *params) {
		fakeFile(p.Fakefs, fakeExePath, 0755, fakeOldVersion)
		p.HttpClient.M[fakeCoderURL+"/api"] = newFakeGetterResponse([]byte{}, 401, variadicS("coder-version: "+fakeNewVersion), nil)
		p.VersionF = func() string { return fakeOldVersion }
		p.ConfirmF = func(string) (string, error) { return "", promptui.ErrAbort }
		u := fromParams(p)
		assertFileContent(t, p.Fakefs, fakeExePath, fakeOldVersion)
		err := u.Run(p.Ctx, false, fakeCoderURL)
		assert.ErrorContains(t, "update coder - user cancelled", err, "failed to confirm update")
		assertFileContent(t, p.Fakefs, fakeExePath, fakeOldVersion)
	})
}

type fakeGetter struct {
	M map[string]*fakeGetterResponse
	T *testing.T
}

func newFakeGetter(t *testing.T) *fakeGetter {
	return &fakeGetter{
		M: make(map[string]*fakeGetterResponse),
	}
}

func (f *fakeGetter) Get(url string) (*http.Response, error) {
	val, ok := f.M[url]
	if !ok {
		f.T.Errorf("unhandled url: %s", url)
		f.T.FailNow()
		return nil, nil // this will never happen
	}
	return val.Resp, val.Err
}

type fakeGetterResponse struct {
	Resp *http.Response
	Err  error
}

func newFakeGetterResponse(body []byte, code int, headers []string, err error) *fakeGetterResponse {
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

	return &fakeGetterResponse{
		Resp: resp,
		Err:  err,
	}
}

func variadicS(s ...string) []string {
	return s
}

// func (f *fakeGetter) Get(url string) (*http.Response, error) {
// 	return f.GetF(url)
// }

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
