package cmd

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"runtime"
	"strings"
	"time"

	"cdr.dev/coder-cli/internal/config"
	"cdr.dev/coder-cli/internal/version"
	"cdr.dev/coder-cli/pkg/clog"
	"golang.org/x/xerrors"

	"github.com/blang/semver/v4"
	"github.com/manifoldco/promptui"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

// updater updates coder-cli.
type updater struct {
	confirmF       func(label string) (string, error)
	executablePath string
	fs             afero.Fs
	httpClient     getter
	osF            func() string
	versionF       func() string
}

func updateCmd() *cobra.Command {
	var (
		force    bool
		coderURL string
	)

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update coder binary",
		Long:  "Update coder to the version matching a given coder instance.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			httpClient := &http.Client{
				Timeout: 10 * time.Second,
			}

			updater := &updater{
				confirmF:       defaultConfirm,
				executablePath: os.Args[0],
				httpClient:     httpClient,
				fs:             afero.NewOsFs(),
				osF:            func() string { return runtime.GOOS },
				versionF:       func() string { return version.Version },
			}
			return updater.Run(ctx, force, coderURL)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "do not prompt for confirmation")
	cmd.Flags().StringVar(&coderURL, "coder", "", "coder instance against which to match version")

	return cmd
}

type getter interface {
	Get(url string) (*http.Response, error)
}

func (u *updater) Run(ctx context.Context, force bool, coderURLString string) error {
	// TODO: check under following directories and warn if coder binary is under them:
	//   * homebrew prefix
	//   * coder assets root (env CODER_ASSETS_ROOT)

	currentBinaryStat, err := u.fs.Stat(u.executablePath)
	if err != nil {
		return clog.Fatal("preflight: cannot stat current binary", clog.Causef("%s", err))
	}

	if currentBinaryStat.Mode().Perm()&0222 == 0 {
		return clog.Fatal("preflight: missing write permission on current binary")
	}

	var coderURL *url.URL
	if coderURLString == "" {
		coderURL, err = getCoderConfigURL()
		if err != nil {
			return clog.Fatal(
				"Unable to automatically determine coder URL",
				clog.Causef(err.Error()),
				clog.BlankLine,
				clog.Tipf("use --coder <url> to specify coder URL"),
			)
		}
	} else {
		coderURL, err = url.Parse(coderURLString)
		if err != nil {
			return clog.Fatal("invalid coder URL", err.Error())
		}
	}

	desiredVersion, err := getAPIVersionUnauthed(u.httpClient, *coderURL)
	if err != nil {
		return clog.Fatal("fetch api version", clog.Causef(err.Error()))
	}

	clog.LogInfo(fmt.Sprintf("Coder instance at %q reports version %s", coderURL.String(), desiredVersion.String()))
	clog.LogInfo(fmt.Sprintf("Current version of coder-cli is %s", version.Version))

	if currentVersion, err := semver.Make(u.versionF()); err == nil {
		if desiredVersion.Compare(currentVersion) == 0 {
			clog.LogInfo("Up to date!")
			return nil
		}
	}

	if !force {
		label := fmt.Sprintf("Update coder-cli to version %s", desiredVersion.FinalizeVersion())
		if _, err := u.confirmF(label); err != nil {
			return clog.Fatal("failed to confirm update", clog.Tipf(`use "--force" to update without confirmation`))
		}
	}

	downloadURL := makeDownloadURL(desiredVersion.FinalizeVersion(), runtime.GOOS, runtime.GOARCH)

	var downloadBuf bytes.Buffer
	memWriter := bufio.NewWriter(&downloadBuf)

	clog.LogInfo("fetching coder-cli from GitHub releases", downloadURL)
	resp, err := u.httpClient.Get(downloadURL)
	if err != nil {
		return clog.Fatal(fmt.Sprintf("failed to fetch URL %s", downloadURL), clog.Causef(err.Error()))
	}

	if resp.StatusCode != http.StatusOK {
		return clog.Fatal("failed to fetch release", clog.Causef("URL %s returned status code %d", downloadURL, resp.StatusCode))
	}

	if _, err := io.Copy(memWriter, resp.Body); err != nil {
		return clog.Fatal(fmt.Sprintf("failed to download %s", downloadURL), clog.Causef(err.Error()))
	}

	_ = resp.Body.Close()

	if err := memWriter.Flush(); err != nil {
		return clog.Fatal(fmt.Sprintf("failed to save %s", downloadURL), clog.Causef(err.Error()))
	}

	// TODO: validate the checksum of the downloaded file. GitHub does not currently provide this information
	// and we do not generate them yet.
	var updatedBinaryName string
	if u.osF() == "windows" {
		updatedBinaryName = "coder.exe"
	} else {
		updatedBinaryName = "coder"
	}
	updatedBinary, err := extractFromArchive(updatedBinaryName, downloadBuf.Bytes())
	if err != nil {
		return clog.Fatal("failed to extract coder binary from archive", clog.Causef(err.Error()))
	}

	// We assume the binary is named coder and write it to coder.new
	updatedCoderBinaryPath := u.executablePath + ".new"
	updatedBin, err := u.fs.OpenFile(updatedCoderBinaryPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, currentBinaryStat.Mode().Perm())
	if err != nil {
		return clog.Fatal("failed to create file for updated coder binary", clog.Causef(err.Error()))
	}

	fsWriter := bufio.NewWriter(updatedBin)
	if _, err := io.Copy(fsWriter, bytes.NewReader(updatedBinary)); err != nil {
		return clog.Fatal("failed to write updated coder binary to disk", clog.Causef(err.Error()))
	}

	if err := fsWriter.Flush(); err != nil {
		return clog.Fatal("failed to persist updated coder binary to disk", clog.Causef(err.Error()))
	}

	if err = u.fs.Rename(updatedCoderBinaryPath, u.executablePath); err != nil {
		return clog.Fatal("failed to update coder binary in-place", clog.Causef(err.Error()))
	}

	clog.LogSuccess("Updated coder CLI to version " + desiredVersion.FinalizeVersion())
	return nil
}

func defaultConfirm(label string) (string, error) {
	p := promptui.Prompt{IsConfirm: true, Label: label}
	return p.Run()
}

func makeDownloadURL(version, ostype, archetype string) string {
	const template = "https://github.com/cdr/coder-cli/releases/download/v%s/coder-cli-%s-%s.%s"
	var ext string
	switch ostype {
	case "linux":
		ext = "tar.gz"
	default:
		ext = ".zip"
	}
	return fmt.Sprintf(template, version, ostype, archetype, ext)
}

func extractFromArchive(path string, archive []byte) ([]byte, error) {
	contentType := http.DetectContentType(archive)
	switch contentType {
	case "application/zip":
		return extractFromZip(path, archive)
	case "application/x-gzip":
		return extractFromTgz(path, archive)
	default:
		return nil, xerrors.Errorf("unknown archive type: %s", contentType)
	}
}

func extractFromZip(path string, archive []byte) ([]byte, error) {
	zipReader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return nil, xerrors.Errorf("failed to open zip archive")
	}

	var zf *zip.File
	for _, f := range zipReader.File {
		if f.Name == path {
			zf = f
			break
		}
	}
	if zf == nil {
		return nil, xerrors.Errorf("could not find path %q in zip archive", path)
	}

	rc, err := zf.Open()
	if err != nil {
		return nil, xerrors.Errorf("failed to extract path %q from archive", path)
	}
	defer rc.Close()

	var b bytes.Buffer
	bw := bufio.NewWriter(&b)
	if _, err := io.Copy(bw, rc); err != nil {
		return nil, xerrors.Errorf("failed to copy path %q to from archive", path)
	}
	return b.Bytes(), nil
}

func extractFromTgz(path string, archive []byte) ([]byte, error) {
	zr, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, xerrors.Errorf("failed to gunzip archive")
	}

	tr := tar.NewReader(zr)

	var b bytes.Buffer
	bw := bufio.NewWriter(&b)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, xerrors.Errorf("failed to read tar archive: %w", err)
		}
		fi := hdr.FileInfo()
		if fi.Name() == path && fi.Mode().IsRegular() {
			_, _ = io.Copy(bw, tr)
			break
		}
	}

	return b.Bytes(), nil
}

// getCoderConfigURL reads the currently configured coder URL, returning an empty string if not configured.
func getCoderConfigURL() (*url.URL, error) {
	urlString, err := config.URL.Read()
	if err != nil {
		return nil, err
	}
	configuredURL, err := url.Parse(strings.TrimSpace(urlString))
	if err != nil {
		return nil, err
	}
	return configuredURL, nil
}

// XXX: coder.Client requires an API key, but we may not be logged into the coder instance for which we
// want to determine the version. We don't need an API key to sniff the version header.
func getAPIVersionUnauthed(client getter, baseURL url.URL) (semver.Version, error) {
	baseURL.Path = path.Join(baseURL.Path, "/api")
	resp, err := client.Get(baseURL.String())
	if err != nil {
		return semver.Version{}, xerrors.Errorf("get %s: %w", baseURL.String(), err)
	}
	defer resp.Body.Close()

	versionHdr := resp.Header.Get("coder-version")
	if versionHdr == "" {
		return semver.Version{}, xerrors.Errorf("URL %s response missing coder-version header", baseURL.String())
	}

	version, err := semver.Parse(versionHdr)
	if err != nil {
		return semver.Version{}, xerrors.Errorf("parsing coder-version header: %w", err)
	}

	return version, nil
}
