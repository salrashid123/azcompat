//go:build go1.18
// +build go1.18

package google

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"golang.org/x/oauth2"
	"google.golang.org/api/idtoken"
)

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type,omitempty"`
	ExpiresIn    int64  `json:"expires_in,omitempty"`
	ExtExpiresIn int64  `json:"ext_expires_in,omitempty"`
}

type GCPAZCredentialsOptions struct {
	azcore.ClientOptions
	ClientID string
	Audience string
	TenantID string
}

type GCPAZCredential struct {
	cred  azcore.TokenCredential
	ts    oauth2.TokenSource
	copts GCPAZCredentialsOptions
}

func NewGCPAZCredentials(options *GCPAZCredentialsOptions) (*GCPAZCredential, error) {
	if options == nil || options.Audience == "" || options.ClientID == "" || options.TenantID == "" {
		return nil, errors.New("Must specify GCPAZCredentialsOptions clientID, Audience and TenantID")
	}

	ctx := context.Background()
	ts, err := idtoken.NewTokenSource(ctx, options.Audience)
	if err != nil {
		return nil, errors.New("could not create gcp oidc token source")
	}
	return &GCPAZCredential{
		ts:    ts,
		copts: *options,
	}, nil
}

func (c *GCPAZCredential) GetToken(ctx context.Context, opts policy.TokenRequestOptions) (azcore.AccessToken, error) {
	tok, err := c.ts.Token()
	if err != nil {
		return azcore.AccessToken{}, err
	}

	// note:  https://stackoverflow.com/questions/62677157/passing-multiple-scope-values-to-oauth-token-endpoint
	///   i'm not sure when and why you'd set multiple scopes in policy.TokenRequestOptions...
	//    anyway, i'm throwing an error if i see more than one

	if len(opts.Scopes) != 1 {
		return azcore.AccessToken{}, errors.New("you must specify precisely one scope")
	}

	stsClient := &http.Client{}
	form := url.Values{}
	form.Add("grant_type", "client_credentials")
	form.Add("scope", opts.Scopes[0])
	form.Add("client_id", c.copts.ClientID)
	form.Add("client_assertion", tok.AccessToken)
	form.Add("client_assertion_type", "urn:ietf:params:oauth:client-assertion-type:jwt-bearer")

	stsResp, err := stsClient.PostForm(fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", c.copts.TenantID), form)
	if err != nil {
		return azcore.AccessToken{}, err
	}
	defer stsResp.Body.Close()

	if stsResp.StatusCode != http.StatusOK {
		bodyBytes, err := ioutil.ReadAll(stsResp.Body)
		if err != nil {
			return azcore.AccessToken{}, err
		}
		return azcore.AccessToken{}, fmt.Errorf("Error reading sts response from azure status %d   %s", stsResp.StatusCode, string(bodyBytes))
	}

	body, err := ioutil.ReadAll(stsResp.Body)
	if err != nil {
		return azcore.AccessToken{}, err
	}

	var result TokenResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return azcore.AccessToken{}, err
	}

	return azcore.AccessToken{
		Token:     result.AccessToken,
		ExpiresOn: time.Unix(result.ExpiresIn, 0),
	}, nil
}
