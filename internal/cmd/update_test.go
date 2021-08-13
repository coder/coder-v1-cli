package cmd

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io/fs"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cdr.dev/slog/sloggers/slogtest/assert"
	"github.com/manifoldco/promptui"
	"github.com/spf13/afero"
)

const (
	fakeExePathLinux      = "/home/user/bin/coder"
	fakeExePathWindows    = `C:\Users\user\bin\coder.exe`
	fakeCoderURL          = "https://my.cdr.dev"
	fakeNewVersion        = "1.23.4-rc.5+678-gabcdef-12345678"
	fakeOldVersion        = "1.22.4-rc.5+678-gabcdef-12345678"
	fakeReleaseURLLinux   = "https://github.com/cdr/coder-cli/releases/download/v1.23.4-rc.5/coder-cli-linux-amd64.tar.gz"
	fakeReleaseURLWindows = "https://github.com/cdr/coder-cli/releases/download/v1.23.4-rc.5/coder-cli-windows-amd64.zip"
)

var (
	apiPrivateVersionURL = fakeCoderURL + "/api/private/version"
	fakeNewVersionJson   = fmt.Sprintf(`{"version":%q}`, fakeNewVersion)
	fakeOldVersionJson   = fmt.Sprintf(`{"version":%q}`, fakeOldVersion)
)

func Test_updater_run(t *testing.T) {
	t.Parallel()

	//  params holds parameters for each test case
	type params struct {
		ConfirmF       func(string) (string, error)
		Ctx            context.Context
		ExecutablePath string
		Fakefs         afero.Fs
		HttpClient     *fakeGetter
		OsF            func() string
		VersionF       func() string
	}

	// fromParams creates a new updater from params
	fromParams := func(p *params) *updater {
		return &updater{
			confirmF:       p.ConfirmF,
			executablePath: p.ExecutablePath,
			fs:             p.Fakefs,
			httpClient:     p.HttpClient,
			osF:            p.OsF,
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
			ExecutablePath: fakeExePathLinux,
			Fakefs:         fakefs,
			HttpClient:     newFakeGetter(t),
			// Default to GOOS=linux
			OsF: func() string { return "linux" },
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
		fakeFile(t, p.Fakefs, fakeExePathLinux, 0755, fakeNewVersion)
		p.HttpClient.M[apiPrivateVersionURL] = newFakeGetterResponse([]byte(fakeNewVersionJson), 200, variadicS(), nil)
		p.VersionF = func() string { return fakeNewVersion }
		u := fromParams(p)
		assertFileContent(t, p.Fakefs, fakeExePathLinux, fakeNewVersion)
		err := u.Run(p.Ctx, false, fakeCoderURL)
		assert.Success(t, "update coder - noop", err)
		assertFileContent(t, p.Fakefs, fakeExePathLinux, fakeNewVersion)
	})

	run(t, "update coder - old to new", func(t *testing.T, p *params) {
		fakeFile(t, p.Fakefs, fakeExePathLinux, 0755, fakeOldVersion)
		p.HttpClient.M[apiPrivateVersionURL] = newFakeGetterResponse([]byte(fakeNewVersionJson), 200, variadicS(), nil)
		p.HttpClient.M[fakeReleaseURLLinux] = newFakeGetterResponse(fakeValidTgzBytes, 200, variadicS(), nil)
		p.VersionF = func() string { return fakeOldVersion }
		p.ConfirmF = fakeConfirmYes
		u := fromParams(p)
		assertFileContent(t, p.Fakefs, fakeExePathLinux, fakeOldVersion)
		err := u.Run(p.Ctx, false, fakeCoderURL)
		assert.Success(t, "update coder - old to new", err)
		assertFileContent(t, p.Fakefs, fakeExePathLinux, fakeNewVersion)
	})

	run(t, "update coder - old to new - binary renamed", func(t *testing.T, p *params) {
		p.ExecutablePath = "/home/user/bin/coder-cli"
		fakeFile(t, p.Fakefs, p.ExecutablePath, 0755, fakeOldVersion)
		p.HttpClient.M[apiPrivateVersionURL] = newFakeGetterResponse([]byte(fakeNewVersionJson), 200, variadicS(), nil)
		p.HttpClient.M[fakeReleaseURLLinux] = newFakeGetterResponse(fakeValidTgzBytes, 200, variadicS(), nil)
		p.VersionF = func() string { return fakeOldVersion }
		p.ConfirmF = fakeConfirmYes
		u := fromParams(p)
		assertFileContent(t, p.Fakefs, p.ExecutablePath, fakeOldVersion)
		err := u.Run(p.Ctx, false, fakeCoderURL)
		assert.Success(t, "update coder - old to new - binary renamed", err)
		assertFileContent(t, p.Fakefs, p.ExecutablePath, fakeNewVersion)
	})

	run(t, "update coder - old to new - windows", func(t *testing.T, p *params) {
		p.OsF = func() string { return "windows" }
		p.ExecutablePath = fakeExePathWindows
		fakeFile(t, p.Fakefs, fakeExePathWindows, 0755, fakeOldVersion)
		p.HttpClient.M[apiPrivateVersionURL] = newFakeGetterResponse([]byte(fakeNewVersionJson), 200, variadicS(), nil)
		p.HttpClient.M[fakeReleaseURLWindows] = newFakeGetterResponse(fakeValidZipBytes, 200, variadicS(), nil)
		p.VersionF = func() string { return fakeOldVersion }
		p.ConfirmF = fakeConfirmYes
		u := fromParams(p)
		assertFileContent(t, p.Fakefs, fakeExePathWindows, fakeOldVersion)
		err := u.Run(p.Ctx, false, fakeCoderURL)
		assert.Success(t, "update coder - old to new - windows", err)
		assertFileContent(t, p.Fakefs, fakeExePathWindows, fakeNewVersion)
	})

	run(t, "update coder - old to new forced", func(t *testing.T, p *params) {
		fakeFile(t, p.Fakefs, fakeExePathLinux, 0755, fakeOldVersion)
		p.HttpClient.M[apiPrivateVersionURL] = newFakeGetterResponse([]byte(fakeNewVersionJson), 200, variadicS(), nil)
		p.HttpClient.M[fakeReleaseURLLinux] = newFakeGetterResponse(fakeValidTgzBytes, 200, variadicS(), nil)
		p.VersionF = func() string { return fakeOldVersion }
		u := fromParams(p)
		assertFileContent(t, p.Fakefs, fakeExePathLinux, fakeOldVersion)
		err := u.Run(p.Ctx, true, fakeCoderURL)
		assert.Success(t, "update coder - old to new forced", err)
		assertFileContent(t, p.Fakefs, fakeExePathLinux, fakeNewVersion)
	})

	run(t, "update coder - user cancelled", func(t *testing.T, p *params) {
		fakeFile(t, p.Fakefs, fakeExePathLinux, 0755, fakeOldVersion)
		p.HttpClient.M[apiPrivateVersionURL] = newFakeGetterResponse([]byte(fakeNewVersionJson), 200, variadicS(), nil)
		p.VersionF = func() string { return fakeOldVersion }
		p.ConfirmF = fakeConfirmNo
		u := fromParams(p)
		assertFileContent(t, p.Fakefs, fakeExePathLinux, fakeOldVersion)
		err := u.Run(p.Ctx, false, fakeCoderURL)
		assert.ErrorContains(t, "update coder - user cancelled", err, "failed to confirm update")
		assertFileContent(t, p.Fakefs, fakeExePathLinux, fakeOldVersion)
	})

	run(t, "update coder - cannot stat", func(t *testing.T, p *params) {
		u := fromParams(p)
		err := u.Run(p.Ctx, false, fakeCoderURL)
		assert.ErrorContains(t, "update coder - cannot stat", err, "cannot stat current binary")
	})

	run(t, "update coder - no permission", func(t *testing.T, p *params) {
		fakeFile(t, p.Fakefs, fakeExePathLinux, 0400, fakeOldVersion)
		u := fromParams(p)
		assertFileContent(t, p.Fakefs, fakeExePathLinux, fakeOldVersion)
		err := u.Run(p.Ctx, false, fakeCoderURL)
		assert.ErrorContains(t, "update coder - no permission", err, "missing write permission")
		assertFileContent(t, p.Fakefs, fakeExePathLinux, fakeOldVersion)
	})

	run(t, "update coder - invalid url", func(t *testing.T, p *params) {
		fakeFile(t, p.Fakefs, fakeExePathLinux, 0755, fakeOldVersion)
		p.VersionF = func() string { return fakeOldVersion }
		u := fromParams(p)
		assertFileContent(t, p.Fakefs, fakeExePathLinux, fakeOldVersion)
		err := u.Run(p.Ctx, false, "h$$p://invalid.url")
		assert.ErrorContains(t, "update coder - invalid url", err, "invalid coder URL")
		assertFileContent(t, p.Fakefs, fakeExePathLinux, fakeOldVersion)
	})

	run(t, "update coder - fetch api version failure", func(t *testing.T, p *params) {
		fakeFile(t, p.Fakefs, fakeExePathLinux, 0755, fakeOldVersion)
		p.HttpClient.M[apiPrivateVersionURL] = newFakeGetterResponse([]byte{}, 401, variadicS(), net.ErrClosed)
		p.VersionF = func() string { return fakeOldVersion }
		u := fromParams(p)
		assertFileContent(t, p.Fakefs, fakeExePathLinux, fakeOldVersion)
		err := u.Run(p.Ctx, false, fakeCoderURL)
		assert.ErrorContains(t, "update coder - fetch api version failure", err, "fetch api version")
		assertFileContent(t, p.Fakefs, fakeExePathLinux, fakeOldVersion)
	})

	run(t, "update coder - failed to fetch URL", func(t *testing.T, p *params) {
		fakeFile(t, p.Fakefs, fakeExePathLinux, 0755, fakeOldVersion)
		p.HttpClient.M[apiPrivateVersionURL] = newFakeGetterResponse([]byte(fakeNewVersionJson), 200, variadicS(), nil)
		p.HttpClient.M[fakeReleaseURLLinux] = newFakeGetterResponse([]byte{}, 0, variadicS(), net.ErrClosed)
		p.VersionF = func() string { return fakeOldVersion }
		p.ConfirmF = fakeConfirmYes
		u := fromParams(p)
		assertFileContent(t, p.Fakefs, fakeExePathLinux, fakeOldVersion)
		err := u.Run(p.Ctx, false, fakeCoderURL)
		assert.ErrorContains(t, "update coder - release URL 404", err, "failed to fetch URL")
		assertFileContent(t, p.Fakefs, fakeExePathLinux, fakeOldVersion)
	})

	run(t, "update coder - release URL 404", func(t *testing.T, p *params) {
		fakeFile(t, p.Fakefs, fakeExePathLinux, 0755, fakeOldVersion)
		p.HttpClient.M[apiPrivateVersionURL] = newFakeGetterResponse([]byte(fakeNewVersionJson), 200, variadicS(), nil)
		p.HttpClient.M[fakeReleaseURLLinux] = newFakeGetterResponse([]byte{}, 404, variadicS(), nil)
		p.VersionF = func() string { return fakeOldVersion }
		p.ConfirmF = fakeConfirmYes
		u := fromParams(p)
		assertFileContent(t, p.Fakefs, fakeExePathLinux, fakeOldVersion)
		err := u.Run(p.Ctx, false, fakeCoderURL)
		assert.ErrorContains(t, "update coder - release URL 404", err, "failed to fetch release")
		assertFileContent(t, p.Fakefs, fakeExePathLinux, fakeOldVersion)
	})

	run(t, "update coder - invalid tgz archive", func(t *testing.T, p *params) {
		fakeFile(t, p.Fakefs, fakeExePathLinux, 0755, fakeOldVersion)
		p.HttpClient.M[apiPrivateVersionURL] = newFakeGetterResponse([]byte(fakeNewVersionJson), 200, variadicS(), nil)
		p.HttpClient.M[fakeReleaseURLLinux] = newFakeGetterResponse([]byte{}, 200, variadicS(), nil)
		p.VersionF = func() string { return fakeOldVersion }
		p.ConfirmF = fakeConfirmYes
		u := fromParams(p)
		assertFileContent(t, p.Fakefs, fakeExePathLinux, fakeOldVersion)
		err := u.Run(p.Ctx, false, fakeCoderURL)
		assert.ErrorContains(t, "update coder - invalid archive", err, "failed to extract coder binary from archive")
		assertFileContent(t, p.Fakefs, fakeExePathLinux, fakeOldVersion)
	})

	run(t, "update coder - invalid zip archive", func(t *testing.T, p *params) {
		p.OsF = func() string { return "windows" }
		p.ExecutablePath = fakeExePathWindows
		fakeFile(t, p.Fakefs, fakeExePathWindows, 0755, fakeOldVersion)
		p.HttpClient.M[apiPrivateVersionURL] = newFakeGetterResponse([]byte(fakeNewVersionJson), 200, variadicS(), nil)
		p.HttpClient.M[fakeReleaseURLWindows] = newFakeGetterResponse([]byte{}, 200, variadicS(), nil)
		p.VersionF = func() string { return fakeOldVersion }
		p.ConfirmF = fakeConfirmYes
		u := fromParams(p)
		assertFileContent(t, p.Fakefs, p.ExecutablePath, fakeOldVersion)
		err := u.Run(p.Ctx, false, fakeCoderURL)
		assert.ErrorContains(t, "update coder - invalid archive", err, "failed to extract coder binary from archive")
		assertFileContent(t, p.Fakefs, p.ExecutablePath, fakeOldVersion)
	})

	run(t, "update coder - read-only fs", func(t *testing.T, p *params) {
		rwfs := p.Fakefs
		p.Fakefs = afero.NewReadOnlyFs(rwfs)
		fakeFile(t, rwfs, fakeExePathLinux, 0755, fakeOldVersion)
		p.HttpClient.M[apiPrivateVersionURL] = newFakeGetterResponse([]byte(fakeNewVersionJson), 200, variadicS(), nil)
		p.HttpClient.M[fakeReleaseURLLinux] = newFakeGetterResponse(fakeValidTgzBytes, 200, variadicS(), nil)
		p.VersionF = func() string { return fakeOldVersion }
		p.ConfirmF = fakeConfirmYes
		u := fromParams(p)
		assertFileContent(t, p.Fakefs, fakeExePathLinux, fakeOldVersion)
		err := u.Run(p.Ctx, false, fakeCoderURL)
		assert.ErrorContains(t, "update coder - read-only fs", err, "failed to create file")
		assertFileContent(t, p.Fakefs, fakeExePathLinux, fakeOldVersion)
	})
}

// fakeGetter mocks HTTP requests
type fakeGetter struct {
	M map[string]*fakeGetterResponse
	T *testing.T
}

func newFakeGetter(t *testing.T) *fakeGetter {
	return &fakeGetter{
		M: make(map[string]*fakeGetterResponse),
		T: t,
	}
}

// Get returns the configured response for url. If no response configured, test fails immediately.
func (f *fakeGetter) Get(url string) (*http.Response, error) {
	f.T.Helper()
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

// newFakeGetterResponse is a convenience function for mocking HTTP requests
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

func fakeConfirmYes(_ string) (string, error) {
	return "y", nil
}

func fakeConfirmNo(_ string) (string, error) {
	return "", promptui.ErrAbort
}

//nolint:unparam
func fakeFile(t *testing.T, fs afero.Fs, name string, perm fs.FileMode, content string) {
	t.Helper()
	err := fs.MkdirAll(filepath.Dir(name), 0750)
	if err != nil {
		panic(err)
	}
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
	t.Helper()
	f, err := fs.OpenFile(name, os.O_RDONLY, 0)
	assert.Success(t, "open file "+name, err)
	defer f.Close()

	b, err := ioutil.ReadAll(f)
	assert.Success(t, "read file "+name, err)

	assert.Equal(t, "assert content equal", content, string(b))
}

// this is a valid tgz archive containing a single file named 'coder' with permissions 0751
// containing the string "1.23.4-rc.5+678-gabcdef-12345678".
var fakeValidTgzBytes, _ = base64.StdEncoding.DecodeString(`H4sIAAAAAAAAA+3QsQ4CIRCEYR6F3oC7wIqvc3KnpQnq+3tGCwsTK3LN/zWTTDWZuG/XeeluJFlV
s1dqNfnOtyJOi4qllHOuTlSTqPMydNXH43afuvfu3w3jb9qExpRjCb1F2x3qMVymU5uXc9CUi63F
1vsAAAAAAAAAAAAAAAAAAL89AYuL424AKAAA`)

// this is a valid zip archive containing a single file named 'coder.exe' with permissions 0751
// containing the string "1.23.4-rc.5+678-gabcdef-12345678".
var fakeValidZipBytes, _ = base64.StdEncoding.DecodeString(`UEsDBAoAAAAAAAtfDVNCHNDCIAAAACAAAAAJABwAY29kZXIuZXhlVVQJAAPmXRZh/10WYXV4CwAB
BOgDAAAE6AMAADEuMjMuNC1yYy41KzY3OC1nYWJjZGVmLTEyMzQ1Njc4UEsBAh4DCgAAAAAAC18N
U0Ic0MIgAAAAIAAAAAkAGAAAAAAAAQAAAO2BAAAAAGNvZGVyLmV4ZVVUBQAD5l0WYXV4CwABBOgD
AAAE6AMAAFBLBQYAAAAAAQABAE8AAABjAAAAAAA=`)
