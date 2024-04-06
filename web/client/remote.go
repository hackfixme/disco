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

	"github.com/mr-tron/base58"
	"go.hackfix.me/disco/web/server/types"
)

// Join sends a request to the remote node to authenticate the local node as a
// client, and allow remote access to store data or admin functionality,
// depending on the permissions granted to the user. The token is generated by
// the server, and is sent in the Authorization header. The pubKey is the
// client's X25519 public key, and is sent in the request body.
// If the token is valid and not expired, the server will generate a TLS client
// certificate, encrypt it with the X25519 shared key, and send it in the
// response body. This method returns the encrypted TLS client certificate and
// client key, and the TLS CA certificate.
func (c *Client) RemoteJoin(ctx context.Context, token, pubKey string) ([]byte, error) {
	url := &url.URL{Scheme: "http", Host: c.address, Path: "/api/v1/join"}

	reqCtx, cancelReqCtx := context.WithCancel(ctx)
	defer cancelReqCtx()

	req, err := http.NewRequestWithContext(
		reqCtx, "POST", url.String(), bytes.NewBufferString(pubKey))
	if err != nil {
		return nil, fmt.Errorf("failed creating request: %w", err)
	}

	req.Header.Set("Authorization", token)

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed sending request: %w", err)
	}
	defer resp.Body.Close()

	joinRespBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed reading response body: %w", err)
	}

	joinResp := &types.RemoteJoinResponse{}
	err = json.Unmarshal(joinRespBody, joinResp)
	if err != nil {
		return nil, fmt.Errorf("failed unmarshalling response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(joinResp.Error)
	}

	joinRespPayloadEnc, err := base58.Decode(joinResp.Data)
	if err != nil {
		return nil, fmt.Errorf("failed decoding response payload: %w", err)
	}

	return joinRespPayloadEnc, nil
}
