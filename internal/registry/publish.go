package registry

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
)

// PublishOptions configures an npm registry publish PUT.
type PublishOptions struct {
	RegistryBase string
	Name         string
	Version      string
	Tag          string
	Access       string
	OTP          string
	AuthToken    string
	Tarball      []byte
	PackageJSON  json.RawMessage
	HTTPClient   *http.Client
}

// PublishResult summarizes a successful publish.
type PublishResult struct {
	URL string
}

// Publish uploads a package tarball via npm-compatible PUT /{name}.
func Publish(ctx context.Context, opts PublishOptions) (PublishResult, error) {
	if err := ctx.Err(); err != nil {
		return PublishResult{}, err
	}
	if opts.Name == "" || opts.Version == "" {
		return PublishResult{}, apperr.New(apperr.Usage, "registry.publish", opts.Name, "name and version are required")
	}
	if len(opts.Tarball) == 0 {
		return PublishResult{}, apperr.New(apperr.Usage, "registry.publish", opts.Name, "tarball is required")
	}
	base := strings.TrimRight(opts.RegistryBase, "/")
	if base == "" {
		return PublishResult{}, apperr.New(apperr.Config, "registry.publish", opts.Name, "registry URL is required")
	}

	tag := opts.Tag
	if tag == "" {
		tag = "latest"
	}
	attachName := tarballAttachmentName(opts.Name, opts.Version)
	body, err := buildPublishBody(opts, attachName, tag)
	if err != nil {
		return PublishResult{}, err
	}

	url := base + "/" + EncodeNamePath(opts.Name)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return PublishResult{}, redactPublishErr(apperr.Wrap(apperr.Network, "registry.publish", opts.Name, err))
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if opts.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+opts.AuthToken)
	}
	if opts.OTP != "" {
		req.Header.Set("npm-otp", opts.OTP)
	}

	hc := opts.HTTPClient
	if hc == nil {
		hc = http.DefaultClient
	}
	res, err := hc.Do(req)
	if err != nil {
		return PublishResult{}, redactPublishErr(apperr.Wrap(apperr.Network, "registry.publish", opts.Name, err))
	}
	defer func() { _ = res.Body.Close() }()
	respBody, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		msg := strings.TrimSpace(string(respBody))
		if msg == "" {
			msg = fmt.Sprintf("registry publish failed (%d)", res.StatusCode)
		}
		return PublishResult{}, redactPublishErr(apperr.New(apperr.Network, "registry.publish", opts.Name, msg))
	}
	return PublishResult{URL: url}, nil
}

func tarballAttachmentName(name, version string) string {
	safe := strings.TrimPrefix(name, "@")
	safe = strings.ReplaceAll(safe, "/", "-")
	return safe + "-" + version + ".tgz"
}

func buildPublishBody(opts PublishOptions, attachName, tag string) ([]byte, error) {
	var meta map[string]any
	if len(opts.PackageJSON) > 0 {
		if err := json.Unmarshal(opts.PackageJSON, &meta); err != nil {
			return nil, apperr.Wrap(apperr.Manifest, "registry.publish", opts.Name, err)
		}
	} else {
		meta = map[string]any{}
	}
	meta["name"] = opts.Name
	meta["version"] = opts.Version

	versionDoc := make(map[string]any, len(meta))
	for k, v := range meta {
		versionDoc[k] = v
	}

	doc := map[string]any{
		"_id":  opts.Name,
		"name": opts.Name,
		"versions": map[string]any{
			opts.Version: versionDoc,
		},
		"dist-tags": map[string]string{
			tag: opts.Version,
		},
		"_attachments": map[string]any{
			attachName: map[string]any{
				"content_type": "application/octet-stream",
				"data":         base64.StdEncoding.EncodeToString(opts.Tarball),
				"length":       len(opts.Tarball),
			},
		},
	}
	if opts.Access != "" {
		doc["access"] = opts.Access
	}
	out, err := json.Marshal(doc)
	if err != nil {
		return nil, apperr.Wrap(apperr.Internal, "registry.publish", opts.Name, err)
	}
	return out, nil
}

func redactPublishErr(err error) error {
	return redactErr(err)
}
