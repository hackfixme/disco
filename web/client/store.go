package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"go.hackfix.me/disco/web/server/types"
)

func (c *Client) StoreGet(ctx context.Context, namespace, key string) (ok bool, value io.Reader, err error) {
	path, err := url.JoinPath("/api/v1/store/value", key)
	if err != nil {
		return false, nil, fmt.Errorf("failed joining URL path: %w", err)
	}
	u := &url.URL{Scheme: "https", Host: c.address, Path: path}

	if namespace != "" {
		q := u.Query()
		q.Set("namespace", namespace)
		qDec, err := url.QueryUnescape(q.Encode())
		if err != nil {
			return false, nil, fmt.Errorf("failed decoding query string: %w", err)
		}
		u.RawQuery = qDec
	}

	reqCtx, cancelReqCtx := context.WithCancel(ctx)
	defer cancelReqCtx()

	req, err := http.NewRequestWithContext(reqCtx, "GET", u.String(), nil)
	if err != nil {
		return false, nil, fmt.Errorf("failed creating request: %w", err)
	}

	resp, err := c.Do(req)
	if err != nil {
		return false, nil, fmt.Errorf("failed sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, nil, fmt.Errorf(
			"request 'GET %s' failed with status %s",
			u.String(), resp.Status)
	}

	var body bytes.Buffer
	io.Copy(&body, resp.Body)

	return true, &body, nil
}

func (c *Client) StoreSet(ctx context.Context, namespace, key string, value io.Reader) error {
	path, err := url.JoinPath("/api/v1/store/value", key)
	if err != nil {
		return fmt.Errorf("failed joining URL path: %w", err)
	}
	u := &url.URL{Scheme: "https", Host: c.address, Path: path}

	if namespace != "" {
		q := u.Query()
		q.Set("namespace", namespace)
		qDec, err := url.QueryUnescape(q.Encode())
		if err != nil {
			return fmt.Errorf("failed decoding query string: %w", err)
		}
		u.RawQuery = qDec
	}

	reqCtx, cancelReqCtx := context.WithCancel(ctx)
	defer cancelReqCtx()

	req, err := http.NewRequestWithContext(reqCtx, "POST", u.String(), value)
	if err != nil {
		return fmt.Errorf("failed creating request: %w", err)
	}

	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("failed sending request: %w", err)
	}
	defer resp.Body.Close()

	setRespBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed reading response body: %w", err)
	}

	setResp := &types.StoreSetResponse{}
	err = json.Unmarshal(setRespBody, setResp)
	if err != nil {
		return fmt.Errorf("failed unmarshalling response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return errors.New(setResp.Error)
	}

	return nil
}
