// Copyright 2022 the Pinniped contributors. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package login

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/client-go/kubernetes/fake"

	"go.pinniped.dev/internal/authenticators"
	"go.pinniped.dev/internal/oidc"
	"go.pinniped.dev/internal/oidc/jwks"
	"go.pinniped.dev/internal/psession"
	"go.pinniped.dev/internal/testutil"
	"go.pinniped.dev/internal/testutil/oidctestutil"
)

func TestPostLoginEndpoint(t *testing.T) {
	const (
		htmlContentType = "text/html; charset=utf-8"

		happyDownstreamCSRF         = "test-csrf"
		happyDownstreamPKCE         = "test-pkce"
		happyDownstreamNonce        = "test-nonce"
		happyDownstreamStateVersion = "2"
		happyEncodedUpstreamState   = "fake-encoded-state-param-value"

		downstreamIssuer              = "https://my-downstream-issuer.com/path"
		downstreamRedirectURI         = "http://127.0.0.1/callback"
		downstreamClientID            = "pinniped-cli"
		happyDownstreamState          = "8b-state"
		downstreamNonce               = "some-nonce-value"
		downstreamPKCEChallenge       = "some-challenge"
		downstreamPKCEChallengeMethod = "S256"

		ldapUpstreamName                   = "some-ldap-idp"
		ldapUpstreamType                   = "ldap"
		ldapUpstreamResourceUID            = "ldap-resource-uid"
		activeDirectoryUpstreamName        = "some-active-directory-idp"
		activeDirectoryUpstreamType        = "activedirectory"
		activeDirectoryUpstreamResourceUID = "active-directory-resource-uid"
		upstreamLDAPURL                    = "ldaps://some-ldap-host:123?base=ou%3Dusers%2Cdc%3Dpinniped%2Cdc%3Ddev"

		userParam                = "username"
		passParam                = "password"
		badUserPassErrParamValue = "login_error"
		internalErrParamValue    = "internal_error"
	)

	var (
		fositeMissingCodeChallengeErrorQuery = map[string]string{
			"error":             "invalid_request",
			"error_description": "The request is missing a required parameter, includes an invalid parameter value, includes a parameter more than once, or is otherwise malformed. Clients must include a code_challenge when performing the authorize code flow, but it is missing.",
			"state":             happyDownstreamState,
		}

		fositeInvalidCodeChallengeErrorQuery = map[string]string{
			"error":             "invalid_request",
			"error_description": "The request is missing a required parameter, includes an invalid parameter value, includes a parameter more than once, or is otherwise malformed. The code_challenge_method is not supported, use S256 instead.",
			"state":             happyDownstreamState,
		}

		fositeMissingCodeChallengeMethodErrorQuery = map[string]string{
			"error":             "invalid_request",
			"error_description": "The request is missing a required parameter, includes an invalid parameter value, includes a parameter more than once, or is otherwise malformed. Clients must use code_challenge_method=S256, plain is not allowed.",
			"state":             happyDownstreamState,
		}

		fositePromptHasNoneAndOtherValueErrorQuery = map[string]string{
			"error":             "invalid_request",
			"error_description": "The request is missing a required parameter, includes an invalid parameter value, includes a parameter more than once, or is otherwise malformed. Parameter 'prompt' was set to 'none', but contains other values as well which is not allowed.",
			"state":             happyDownstreamState,
		}
	)

	happyDownstreamScopesRequested := []string{"openid"}
	happyDownstreamScopesGranted := []string{"openid"}

	happyDownstreamRequestParamsQuery := url.Values{
		"response_type":         []string{"code"},
		"scope":                 []string{strings.Join(happyDownstreamScopesRequested, " ")},
		"client_id":             []string{downstreamClientID},
		"state":                 []string{happyDownstreamState},
		"nonce":                 []string{downstreamNonce},
		"code_challenge":        []string{downstreamPKCEChallenge},
		"code_challenge_method": []string{downstreamPKCEChallengeMethod},
		"redirect_uri":          []string{downstreamRedirectURI},
	}
	happyDownstreamRequestParams := happyDownstreamRequestParamsQuery.Encode()

	copyOfHappyDownstreamRequestParamsQuery := func() url.Values {
		params := url.Values{}
		for k, v := range happyDownstreamRequestParamsQuery {
			params[k] = make([]string, len(v))
			copy(params[k], v)
		}
		return params
	}

	happyLDAPDecodedState := &oidc.UpstreamStateParamData{
		AuthParams:    happyDownstreamRequestParams,
		UpstreamName:  ldapUpstreamName,
		UpstreamType:  ldapUpstreamType,
		Nonce:         happyDownstreamNonce,
		CSRFToken:     happyDownstreamCSRF,
		PKCECode:      happyDownstreamPKCE,
		FormatVersion: happyDownstreamStateVersion,
	}

	modifyHappyLDAPDecodedState := func(edit func(*oidc.UpstreamStateParamData)) *oidc.UpstreamStateParamData {
		copyOfHappyLDAPDecodedState := *happyLDAPDecodedState
		edit(&copyOfHappyLDAPDecodedState)
		return &copyOfHappyLDAPDecodedState
	}

	happyActiveDirectoryDecodedState := &oidc.UpstreamStateParamData{
		AuthParams:    happyDownstreamRequestParams,
		UpstreamName:  activeDirectoryUpstreamName,
		UpstreamType:  activeDirectoryUpstreamType,
		Nonce:         happyDownstreamNonce,
		CSRFToken:     happyDownstreamCSRF,
		PKCECode:      happyDownstreamPKCE,
		FormatVersion: happyDownstreamStateVersion,
	}

	happyLDAPUsername := "some-ldap-user"
	happyLDAPUsernameFromAuthenticator := "some-mapped-ldap-username"
	happyLDAPPassword := "some-ldap-password" //nolint:gosec
	happyLDAPUID := "some-ldap-uid"
	happyLDAPUserDN := "cn=foo,dn=bar"
	happyLDAPGroups := []string{"group1", "group2", "group3"}
	happyLDAPExtraRefreshAttribute := "some-refresh-attribute"
	happyLDAPExtraRefreshValue := "some-refresh-attribute-value"

	parsedUpstreamLDAPURL, err := url.Parse(upstreamLDAPURL)
	require.NoError(t, err)

	ldapAuthenticateFunc := func(ctx context.Context, username, password string) (*authenticators.Response, bool, error) {
		if username == "" || password == "" {
			return nil, false, fmt.Errorf("should not have passed empty username or password to the authenticator")
		}
		if username == happyLDAPUsername && password == happyLDAPPassword {
			return &authenticators.Response{
				User: &user.DefaultInfo{
					Name:   happyLDAPUsernameFromAuthenticator,
					UID:    happyLDAPUID,
					Groups: happyLDAPGroups,
				},
				DN: happyLDAPUserDN,
				ExtraRefreshAttributes: map[string]string{
					happyLDAPExtraRefreshAttribute: happyLDAPExtraRefreshValue,
				},
			}, true, nil
		}
		return nil, false, nil
	}

	upstreamLDAPIdentityProvider := oidctestutil.TestUpstreamLDAPIdentityProvider{
		Name:             ldapUpstreamName,
		ResourceUID:      ldapUpstreamResourceUID,
		URL:              parsedUpstreamLDAPURL,
		AuthenticateFunc: ldapAuthenticateFunc,
	}

	upstreamActiveDirectoryIdentityProvider := oidctestutil.TestUpstreamLDAPIdentityProvider{
		Name:             activeDirectoryUpstreamName,
		ResourceUID:      activeDirectoryUpstreamResourceUID,
		URL:              parsedUpstreamLDAPURL,
		AuthenticateFunc: ldapAuthenticateFunc,
	}

	erroringUpstreamLDAPIdentityProvider := oidctestutil.TestUpstreamLDAPIdentityProvider{
		Name:        ldapUpstreamName,
		ResourceUID: ldapUpstreamResourceUID,
		AuthenticateFunc: func(ctx context.Context, username, password string) (*authenticators.Response, bool, error) {
			return nil, false, fmt.Errorf("some ldap upstream auth error")
		},
	}

	expectedHappyActiveDirectoryUpstreamCustomSession := &psession.CustomSessionData{
		ProviderUID:  activeDirectoryUpstreamResourceUID,
		ProviderName: activeDirectoryUpstreamName,
		ProviderType: psession.ProviderTypeActiveDirectory,
		OIDC:         nil,
		LDAP:         nil,
		ActiveDirectory: &psession.ActiveDirectorySessionData{
			UserDN:                 happyLDAPUserDN,
			ExtraRefreshAttributes: map[string]string{happyLDAPExtraRefreshAttribute: happyLDAPExtraRefreshValue},
		},
	}

	expectedHappyLDAPUpstreamCustomSession := &psession.CustomSessionData{
		ProviderUID:  ldapUpstreamResourceUID,
		ProviderName: ldapUpstreamName,
		ProviderType: psession.ProviderTypeLDAP,
		OIDC:         nil,
		LDAP: &psession.LDAPSessionData{
			UserDN:                 happyLDAPUserDN,
			ExtraRefreshAttributes: map[string]string{happyLDAPExtraRefreshAttribute: happyLDAPExtraRefreshValue},
		},
		ActiveDirectory: nil,
	}

	// Note that fosite puts the granted scopes as a param in the redirect URI even though the spec doesn't seem to require it
	happyAuthcodeDownstreamRedirectLocationRegexp := downstreamRedirectURI + `\?code=([^&]+)&scope=openid&state=` + happyDownstreamState

	happyUsernamePasswordFormParams := url.Values{userParam: []string{happyLDAPUsername}, passParam: []string{happyLDAPPassword}}

	encodeQuery := func(query map[string]string) string {
		values := url.Values{}
		for k, v := range query {
			values[k] = []string{v}
		}
		return values.Encode()
	}

	urlWithQuery := func(baseURL string, query map[string]string) string {
		urlToReturn := fmt.Sprintf("%s?%s", baseURL, encodeQuery(query))
		_, err := url.Parse(urlToReturn)
		require.NoError(t, err, "urlWithQuery helper was used to create an illegal URL")
		return urlToReturn
	}

	tests := []struct {
		name         string
		idps         *oidctestutil.UpstreamIDPListerBuilder
		decodedState *oidc.UpstreamStateParamData
		formParams   url.Values
		reqURIQuery  url.Values

		wantStatus      int
		wantContentType string
		wantBodyString  string
		wantErr         string

		// Assertion that the response should be a redirect to the login page with an error param.
		wantRedirectToLoginPageError string

		// Assertions for when an authcode should be returned, i.e. the request was authenticated by an
		// upstream LDAP or AD provider.
		wantRedirectLocationRegexp        string // for loose matching
		wantRedirectLocationString        string // for exact matching instead
		wantBodyFormResponseRegexp        string // for form_post html page matching instead
		wantDownstreamRedirectURI         string
		wantDownstreamGrantedScopes       []string
		wantDownstreamIDTokenSubject      string
		wantDownstreamIDTokenUsername     string
		wantDownstreamIDTokenGroups       []string
		wantDownstreamRequestedScopes     []string
		wantDownstreamPKCEChallenge       string
		wantDownstreamPKCEChallengeMethod string
		wantDownstreamNonce               string
		wantDownstreamCustomSessionData   *psession.CustomSessionData

		// Authorization requests for either a successful OIDC upstream or for an error with any upstream
		// should never use Kube storage. There is only one exception to this rule, which is that certain
		// OIDC validations are checked in fosite after the OAuth authcode (and sometimes the OIDC session)
		// is stored, so it is possible with an LDAP upstream to store objects and then return an error to
		// the client anyway (which makes the stored objects useless, but oh well).
		wantUnnecessaryStoredRecords int
	}{
		{
			name: "happy LDAP login",
			idps: oidctestutil.NewUpstreamIDPListerBuilder().
				WithLDAP(&upstreamLDAPIdentityProvider). // should pick this one
				WithActiveDirectory(&erroringUpstreamLDAPIdentityProvider),
			decodedState:                      happyLDAPDecodedState,
			formParams:                        happyUsernamePasswordFormParams,
			wantStatus:                        http.StatusSeeOther,
			wantContentType:                   htmlContentType,
			wantBodyString:                    "",
			wantRedirectLocationRegexp:        happyAuthcodeDownstreamRedirectLocationRegexp,
			wantDownstreamIDTokenSubject:      upstreamLDAPURL + "&sub=" + happyLDAPUID,
			wantDownstreamIDTokenUsername:     happyLDAPUsernameFromAuthenticator,
			wantDownstreamIDTokenGroups:       happyLDAPGroups,
			wantDownstreamRequestedScopes:     happyDownstreamScopesRequested,
			wantDownstreamRedirectURI:         downstreamRedirectURI,
			wantDownstreamGrantedScopes:       happyDownstreamScopesGranted,
			wantDownstreamNonce:               downstreamNonce,
			wantDownstreamPKCEChallenge:       downstreamPKCEChallenge,
			wantDownstreamPKCEChallengeMethod: downstreamPKCEChallengeMethod,
			wantDownstreamCustomSessionData:   expectedHappyLDAPUpstreamCustomSession,
		},
		{
			name: "happy AD login",
			idps: oidctestutil.NewUpstreamIDPListerBuilder().
				WithLDAP(&erroringUpstreamLDAPIdentityProvider).
				WithActiveDirectory(&upstreamActiveDirectoryIdentityProvider), // should pick this one
			decodedState:                      happyActiveDirectoryDecodedState,
			formParams:                        happyUsernamePasswordFormParams,
			wantStatus:                        http.StatusSeeOther,
			wantContentType:                   htmlContentType,
			wantBodyString:                    "",
			wantRedirectLocationRegexp:        happyAuthcodeDownstreamRedirectLocationRegexp,
			wantDownstreamIDTokenSubject:      upstreamLDAPURL + "&sub=" + happyLDAPUID,
			wantDownstreamIDTokenUsername:     happyLDAPUsernameFromAuthenticator,
			wantDownstreamIDTokenGroups:       happyLDAPGroups,
			wantDownstreamRequestedScopes:     happyDownstreamScopesRequested,
			wantDownstreamRedirectURI:         downstreamRedirectURI,
			wantDownstreamGrantedScopes:       happyDownstreamScopesGranted,
			wantDownstreamNonce:               downstreamNonce,
			wantDownstreamPKCEChallenge:       downstreamPKCEChallenge,
			wantDownstreamPKCEChallengeMethod: downstreamPKCEChallengeMethod,
			wantDownstreamCustomSessionData:   expectedHappyActiveDirectoryUpstreamCustomSession,
		},
		{
			name: "happy LDAP login when downstream response_mode=form_post returns 200 with HTML+JS form",
			idps: oidctestutil.NewUpstreamIDPListerBuilder().WithLDAP(&upstreamLDAPIdentityProvider),
			decodedState: modifyHappyLDAPDecodedState(func(data *oidc.UpstreamStateParamData) {
				query := copyOfHappyDownstreamRequestParamsQuery()
				query["response_mode"] = []string{"form_post"}
				data.AuthParams = query.Encode()
			}),
			formParams:      happyUsernamePasswordFormParams,
			wantStatus:      http.StatusOK,
			wantContentType: htmlContentType,
			wantBodyFormResponseRegexp: `(?s)<html.*<script>.*To finish logging in, paste this authorization code` +
				`.*<form>.*<code id="manual-auth-code">(.+)</code>.*</html>`, // "(?s)" means match "." across newlines
			wantDownstreamIDTokenSubject:      upstreamLDAPURL + "&sub=" + happyLDAPUID,
			wantDownstreamIDTokenUsername:     happyLDAPUsernameFromAuthenticator,
			wantDownstreamIDTokenGroups:       happyLDAPGroups,
			wantDownstreamRequestedScopes:     happyDownstreamScopesRequested,
			wantDownstreamRedirectURI:         downstreamRedirectURI,
			wantDownstreamGrantedScopes:       happyDownstreamScopesGranted,
			wantDownstreamNonce:               downstreamNonce,
			wantDownstreamPKCEChallenge:       downstreamPKCEChallenge,
			wantDownstreamPKCEChallengeMethod: downstreamPKCEChallengeMethod,
			wantDownstreamCustomSessionData:   expectedHappyLDAPUpstreamCustomSession,
		},
		{
			name: "happy LDAP login when downstream redirect uri matches what is configured for client except for the port number",
			idps: oidctestutil.NewUpstreamIDPListerBuilder().WithLDAP(&upstreamLDAPIdentityProvider),
			decodedState: modifyHappyLDAPDecodedState(func(data *oidc.UpstreamStateParamData) {
				query := copyOfHappyDownstreamRequestParamsQuery()
				query["redirect_uri"] = []string{"http://127.0.0.1:4242/callback"}
				data.AuthParams = query.Encode()
			}),
			formParams:                        happyUsernamePasswordFormParams,
			wantStatus:                        http.StatusSeeOther,
			wantContentType:                   htmlContentType,
			wantBodyString:                    "",
			wantRedirectLocationRegexp:        "http://127.0.0.1:4242/callback" + `\?code=([^&]+)&scope=openid&state=` + happyDownstreamState,
			wantDownstreamIDTokenSubject:      upstreamLDAPURL + "&sub=" + happyLDAPUID,
			wantDownstreamIDTokenUsername:     happyLDAPUsernameFromAuthenticator,
			wantDownstreamIDTokenGroups:       happyLDAPGroups,
			wantDownstreamRequestedScopes:     happyDownstreamScopesRequested,
			wantDownstreamRedirectURI:         "http://127.0.0.1:4242/callback",
			wantDownstreamGrantedScopes:       happyDownstreamScopesGranted,
			wantDownstreamNonce:               downstreamNonce,
			wantDownstreamPKCEChallenge:       downstreamPKCEChallenge,
			wantDownstreamPKCEChallengeMethod: downstreamPKCEChallengeMethod,
			wantDownstreamCustomSessionData:   expectedHappyLDAPUpstreamCustomSession,
		},
		{
			name: "happy LDAP login when there are additional allowed downstream requested scopes",
			idps: oidctestutil.NewUpstreamIDPListerBuilder().WithLDAP(&upstreamLDAPIdentityProvider),
			decodedState: modifyHappyLDAPDecodedState(func(data *oidc.UpstreamStateParamData) {
				query := copyOfHappyDownstreamRequestParamsQuery()
				query["scope"] = []string{"openid offline_access pinniped:request-audience"}
				data.AuthParams = query.Encode()
			}),
			formParams:                        happyUsernamePasswordFormParams,
			wantStatus:                        http.StatusSeeOther,
			wantContentType:                   htmlContentType,
			wantBodyString:                    "",
			wantRedirectLocationRegexp:        downstreamRedirectURI + `\?code=([^&]+)&scope=openid\+offline_access\+pinniped%3Arequest-audience&state=` + happyDownstreamState,
			wantDownstreamIDTokenSubject:      upstreamLDAPURL + "&sub=" + happyLDAPUID,
			wantDownstreamIDTokenUsername:     happyLDAPUsernameFromAuthenticator,
			wantDownstreamIDTokenGroups:       happyLDAPGroups,
			wantDownstreamRequestedScopes:     []string{"openid", "offline_access", "pinniped:request-audience"},
			wantDownstreamRedirectURI:         downstreamRedirectURI,
			wantDownstreamGrantedScopes:       []string{"openid", "offline_access", "pinniped:request-audience"},
			wantDownstreamNonce:               downstreamNonce,
			wantDownstreamPKCEChallenge:       downstreamPKCEChallenge,
			wantDownstreamPKCEChallengeMethod: downstreamPKCEChallengeMethod,
			wantDownstreamCustomSessionData:   expectedHappyLDAPUpstreamCustomSession,
		},
		{
			name: "happy LDAP when downstream OIDC validations are skipped because the openid scope was not requested",
			idps: oidctestutil.NewUpstreamIDPListerBuilder().WithLDAP(&upstreamLDAPIdentityProvider),
			decodedState: modifyHappyLDAPDecodedState(func(data *oidc.UpstreamStateParamData) {
				query := copyOfHappyDownstreamRequestParamsQuery()
				query["scope"] = []string{"email"}
				// The following prompt value is illegal when openid is requested, but note that openid is not requested.
				query["prompt"] = []string{"none login"}
				data.AuthParams = query.Encode()
			}),
			formParams:                        happyUsernamePasswordFormParams,
			wantStatus:                        http.StatusSeeOther,
			wantContentType:                   htmlContentType,
			wantBodyString:                    "",
			wantRedirectLocationRegexp:        downstreamRedirectURI + `\?code=([^&]+)&scope=&state=` + happyDownstreamState, // no scopes granted
			wantDownstreamIDTokenSubject:      upstreamLDAPURL + "&sub=" + happyLDAPUID,
			wantDownstreamIDTokenUsername:     happyLDAPUsernameFromAuthenticator,
			wantDownstreamIDTokenGroups:       happyLDAPGroups,
			wantDownstreamRequestedScopes:     []string{"email"}, // only email was requested
			wantDownstreamRedirectURI:         downstreamRedirectURI,
			wantDownstreamGrantedScopes:       []string{}, // no scopes granted
			wantDownstreamNonce:               downstreamNonce,
			wantDownstreamPKCEChallenge:       downstreamPKCEChallenge,
			wantDownstreamPKCEChallengeMethod: downstreamPKCEChallengeMethod,
			wantDownstreamCustomSessionData:   expectedHappyLDAPUpstreamCustomSession,
		},
		{
			name:                         "bad username LDAP login",
			idps:                         oidctestutil.NewUpstreamIDPListerBuilder().WithLDAP(&upstreamLDAPIdentityProvider),
			decodedState:                 happyLDAPDecodedState,
			formParams:                   url.Values{userParam: []string{"wrong!"}, passParam: []string{happyLDAPPassword}},
			wantStatus:                   http.StatusSeeOther,
			wantContentType:              htmlContentType,
			wantBodyString:               "",
			wantRedirectToLoginPageError: badUserPassErrParamValue,
		},
		{
			name:                         "bad password LDAP login",
			idps:                         oidctestutil.NewUpstreamIDPListerBuilder().WithLDAP(&upstreamLDAPIdentityProvider),
			decodedState:                 happyLDAPDecodedState,
			formParams:                   url.Values{userParam: []string{happyLDAPUsername}, passParam: []string{"wrong!"}},
			wantStatus:                   http.StatusSeeOther,
			wantContentType:              htmlContentType,
			wantBodyString:               "",
			wantRedirectToLoginPageError: badUserPassErrParamValue,
		},
		{
			name:                         "blank username LDAP login",
			idps:                         oidctestutil.NewUpstreamIDPListerBuilder().WithLDAP(&upstreamLDAPIdentityProvider),
			decodedState:                 happyLDAPDecodedState,
			formParams:                   url.Values{userParam: []string{""}, passParam: []string{happyLDAPPassword}},
			wantStatus:                   http.StatusSeeOther,
			wantContentType:              htmlContentType,
			wantBodyString:               "",
			wantRedirectToLoginPageError: badUserPassErrParamValue,
		},
		{
			name:                         "blank password LDAP login",
			idps:                         oidctestutil.NewUpstreamIDPListerBuilder().WithLDAP(&upstreamLDAPIdentityProvider),
			decodedState:                 happyLDAPDecodedState,
			formParams:                   url.Values{userParam: []string{happyLDAPUsername}, passParam: []string{""}},
			wantStatus:                   http.StatusSeeOther,
			wantContentType:              htmlContentType,
			wantBodyString:               "",
			wantRedirectToLoginPageError: badUserPassErrParamValue,
		},
		{
			name:                         "username and password sent as URI query params should be ignored since they are expected in form post body",
			idps:                         oidctestutil.NewUpstreamIDPListerBuilder().WithLDAP(&upstreamLDAPIdentityProvider),
			decodedState:                 happyLDAPDecodedState,
			reqURIQuery:                  happyUsernamePasswordFormParams,
			wantStatus:                   http.StatusSeeOther,
			wantContentType:              htmlContentType,
			wantBodyString:               "",
			wantRedirectToLoginPageError: badUserPassErrParamValue,
		},
		{
			name:                         "error during upstream LDAP authentication",
			idps:                         oidctestutil.NewUpstreamIDPListerBuilder().WithLDAP(&erroringUpstreamLDAPIdentityProvider),
			decodedState:                 happyLDAPDecodedState,
			formParams:                   happyUsernamePasswordFormParams,
			wantStatus:                   http.StatusSeeOther,
			wantContentType:              htmlContentType,
			wantBodyString:               "",
			wantRedirectToLoginPageError: internalErrParamValue,
		},
		{
			name: "downstream redirect uri does not match what is configured for client",
			idps: oidctestutil.NewUpstreamIDPListerBuilder().WithLDAP(&upstreamLDAPIdentityProvider),
			decodedState: modifyHappyLDAPDecodedState(func(data *oidc.UpstreamStateParamData) {
				query := copyOfHappyDownstreamRequestParamsQuery()
				query["redirect_uri"] = []string{"http://127.0.0.1/wrong_callback"}
				data.AuthParams = query.Encode()
			}),
			formParams: happyUsernamePasswordFormParams,
			wantErr:    "error using state downstream auth params",
		},
		{
			name: "downstream client does not exist",
			idps: oidctestutil.NewUpstreamIDPListerBuilder().WithLDAP(&upstreamLDAPIdentityProvider),
			decodedState: modifyHappyLDAPDecodedState(func(data *oidc.UpstreamStateParamData) {
				query := copyOfHappyDownstreamRequestParamsQuery()
				query["client_id"] = []string{"wrong_client_id"}
				data.AuthParams = query.Encode()
			}),
			formParams: happyUsernamePasswordFormParams,
			wantErr:    "error using state downstream auth params",
		},
		{
			name: "downstream client is missing",
			idps: oidctestutil.NewUpstreamIDPListerBuilder().WithLDAP(&upstreamLDAPIdentityProvider),
			decodedState: modifyHappyLDAPDecodedState(func(data *oidc.UpstreamStateParamData) {
				query := copyOfHappyDownstreamRequestParamsQuery()
				delete(query, "client_id")
				data.AuthParams = query.Encode()
			}),
			formParams: happyUsernamePasswordFormParams,
			wantErr:    "error using state downstream auth params",
		},
		{
			name: "response type is unsupported",
			idps: oidctestutil.NewUpstreamIDPListerBuilder().WithLDAP(&upstreamLDAPIdentityProvider),
			decodedState: modifyHappyLDAPDecodedState(func(data *oidc.UpstreamStateParamData) {
				query := copyOfHappyDownstreamRequestParamsQuery()
				query["response_type"] = []string{"unsupported"}
				data.AuthParams = query.Encode()
			}),
			formParams: happyUsernamePasswordFormParams,
			wantErr:    "error using state downstream auth params",
		},
		{
			name: "response type is missing",
			idps: oidctestutil.NewUpstreamIDPListerBuilder().WithLDAP(&upstreamLDAPIdentityProvider),
			decodedState: modifyHappyLDAPDecodedState(func(data *oidc.UpstreamStateParamData) {
				query := copyOfHappyDownstreamRequestParamsQuery()
				delete(query, "response_type")
				data.AuthParams = query.Encode()
			}),
			formParams: happyUsernamePasswordFormParams,
			wantErr:    "error using state downstream auth params",
		},
		{
			name: "PKCE code_challenge is missing",
			idps: oidctestutil.NewUpstreamIDPListerBuilder().WithLDAP(&upstreamLDAPIdentityProvider),
			decodedState: modifyHappyLDAPDecodedState(func(data *oidc.UpstreamStateParamData) {
				query := copyOfHappyDownstreamRequestParamsQuery()
				delete(query, "code_challenge")
				data.AuthParams = query.Encode()
			}),
			formParams:                   happyUsernamePasswordFormParams,
			wantStatus:                   http.StatusSeeOther,
			wantContentType:              htmlContentType,
			wantBodyString:               "",
			wantRedirectLocationString:   urlWithQuery(downstreamRedirectURI, fositeMissingCodeChallengeErrorQuery),
			wantUnnecessaryStoredRecords: 2, // fosite already stored the authcode and oidc session before it noticed the error
		},
		{
			name: "PKCE code_challenge_method is invalid",
			idps: oidctestutil.NewUpstreamIDPListerBuilder().WithLDAP(&upstreamLDAPIdentityProvider),
			decodedState: modifyHappyLDAPDecodedState(func(data *oidc.UpstreamStateParamData) {
				query := copyOfHappyDownstreamRequestParamsQuery()
				query["code_challenge_method"] = []string{"this-is-not-a-valid-pkce-alg"}
				data.AuthParams = query.Encode()
			}),
			formParams:                   happyUsernamePasswordFormParams,
			wantStatus:                   http.StatusSeeOther,
			wantContentType:              htmlContentType,
			wantBodyString:               "",
			wantRedirectLocationString:   urlWithQuery(downstreamRedirectURI, fositeInvalidCodeChallengeErrorQuery),
			wantUnnecessaryStoredRecords: 2, // fosite already stored the authcode and oidc session before it noticed the error
		},
		{
			name: "PKCE code_challenge_method is `plain`",
			idps: oidctestutil.NewUpstreamIDPListerBuilder().WithLDAP(&upstreamLDAPIdentityProvider),
			decodedState: modifyHappyLDAPDecodedState(func(data *oidc.UpstreamStateParamData) {
				query := copyOfHappyDownstreamRequestParamsQuery()
				query["code_challenge_method"] = []string{"plain"} // plain is not allowed
				data.AuthParams = query.Encode()
			}),
			formParams:                   happyUsernamePasswordFormParams,
			wantStatus:                   http.StatusSeeOther,
			wantContentType:              htmlContentType,
			wantBodyString:               "",
			wantRedirectLocationString:   urlWithQuery(downstreamRedirectURI, fositeMissingCodeChallengeMethodErrorQuery),
			wantUnnecessaryStoredRecords: 2, // fosite already stored the authcode and oidc session before it noticed the error
		},
		{
			name: "PKCE code_challenge_method is missing",
			idps: oidctestutil.NewUpstreamIDPListerBuilder().WithLDAP(&upstreamLDAPIdentityProvider),
			decodedState: modifyHappyLDAPDecodedState(func(data *oidc.UpstreamStateParamData) {
				query := copyOfHappyDownstreamRequestParamsQuery()
				delete(query, "code_challenge_method")
				data.AuthParams = query.Encode()
			}),
			formParams:                   happyUsernamePasswordFormParams,
			wantStatus:                   http.StatusSeeOther,
			wantContentType:              htmlContentType,
			wantBodyString:               "",
			wantRedirectLocationString:   urlWithQuery(downstreamRedirectURI, fositeMissingCodeChallengeMethodErrorQuery),
			wantUnnecessaryStoredRecords: 2, // fosite already stored the authcode and oidc session before it noticed the error
		},
		{
			name: "prompt param is not allowed to have none and another legal value at the same time",
			idps: oidctestutil.NewUpstreamIDPListerBuilder().WithLDAP(&upstreamLDAPIdentityProvider),
			decodedState: modifyHappyLDAPDecodedState(func(data *oidc.UpstreamStateParamData) {
				query := copyOfHappyDownstreamRequestParamsQuery()
				query["prompt"] = []string{"none login"}
				data.AuthParams = query.Encode()
			}),
			formParams:                   happyUsernamePasswordFormParams,
			wantStatus:                   http.StatusSeeOther,
			wantContentType:              htmlContentType,
			wantBodyString:               "",
			wantRedirectLocationString:   urlWithQuery(downstreamRedirectURI, fositePromptHasNoneAndOtherValueErrorQuery),
			wantUnnecessaryStoredRecords: 1, // fosite already stored the authcode before it noticed the error
		},
		{
			name: "downstream state does not have enough entropy",
			idps: oidctestutil.NewUpstreamIDPListerBuilder().WithLDAP(&upstreamLDAPIdentityProvider),
			decodedState: modifyHappyLDAPDecodedState(func(data *oidc.UpstreamStateParamData) {
				query := copyOfHappyDownstreamRequestParamsQuery()
				query["state"] = []string{"short"}
				data.AuthParams = query.Encode()
			}),
			formParams: happyUsernamePasswordFormParams,
			wantErr:    "error using state downstream auth params",
		},
		{
			name: "downstream scopes do not match what is configured for client",
			idps: oidctestutil.NewUpstreamIDPListerBuilder().WithLDAP(&upstreamLDAPIdentityProvider),
			decodedState: modifyHappyLDAPDecodedState(func(data *oidc.UpstreamStateParamData) {
				query := copyOfHappyDownstreamRequestParamsQuery()
				query["scope"] = []string{"openid offline_access pinniped:request-audience scope_not_allowed"}
				data.AuthParams = query.Encode()
			}),
			formParams: happyUsernamePasswordFormParams,
			wantErr:    "error using state downstream auth params",
		},
		{
			name:         "no upstream providers are configured or provider cannot be found by name",
			idps:         oidctestutil.NewUpstreamIDPListerBuilder(), // empty
			decodedState: happyLDAPDecodedState,
			formParams:   happyUsernamePasswordFormParams,
			wantErr:      "error finding upstream provider: provider not found",
		},
		{
			name:         "upstream provider cannot be found by name and type",
			idps:         oidctestutil.NewUpstreamIDPListerBuilder().WithLDAP(&upstreamLDAPIdentityProvider),
			decodedState: happyActiveDirectoryDecodedState, // correct upstream IDP name, but wrong upstream IDP type
			formParams:   happyUsernamePasswordFormParams,
			wantErr:      "error finding upstream provider: provider not found",
		},
	}

	for _, test := range tests {
		tt := test

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			kubeClient := fake.NewSimpleClientset()
			secretsClient := kubeClient.CoreV1().Secrets("some-namespace")

			// Configure fosite the same way that the production code would.
			// Inject this into our test subject at the last second so we get a fresh storage for every test.
			timeoutsConfiguration := oidc.DefaultOIDCTimeoutsConfiguration()
			kubeOauthStore := oidc.NewKubeStorage(secretsClient, timeoutsConfiguration)
			hmacSecretFunc := func() []byte { return []byte("some secret - must have at least 32 bytes") }
			require.GreaterOrEqual(t, len(hmacSecretFunc()), 32, "fosite requires that hmac secrets have at least 32 bytes")
			jwksProviderIsUnused := jwks.NewDynamicJWKSProvider()
			oauthHelper := oidc.FositeOauth2Helper(kubeOauthStore, downstreamIssuer, hmacSecretFunc, jwksProviderIsUnused, timeoutsConfiguration)

			req := httptest.NewRequest(http.MethodPost, "/ignored", strings.NewReader(tt.formParams.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			if tt.reqURIQuery != nil {
				req.URL.RawQuery = tt.reqURIQuery.Encode()
			}

			rsp := httptest.NewRecorder()

			subject := NewPostHandler(downstreamIssuer, tt.idps.Build(), oauthHelper)

			err := subject(rsp, req, happyEncodedUpstreamState, tt.decodedState)
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				require.Empty(t, kubeClient.Actions())
				return // the http response doesn't matter when the function returns an error, because the caller should handle the error
			}
			// Otherwise, expect no error.
			require.NoError(t, err)

			require.Equal(t, tt.wantStatus, rsp.Code)
			testutil.RequireEqualContentType(t, rsp.Header().Get("Content-Type"), tt.wantContentType)

			actualLocation := rsp.Header().Get("Location")

			switch {
			case tt.wantRedirectLocationRegexp != "":
				// Expecting a success redirect to the client.
				require.Equal(t, tt.wantBodyString, rsp.Body.String())
				require.Len(t, rsp.Header().Values("Location"), 1)
				oidctestutil.RequireAuthCodeRegexpMatch(
					t,
					actualLocation,
					tt.wantRedirectLocationRegexp,
					kubeClient,
					secretsClient,
					kubeOauthStore,
					tt.wantDownstreamGrantedScopes,
					tt.wantDownstreamIDTokenSubject,
					tt.wantDownstreamIDTokenUsername,
					tt.wantDownstreamIDTokenGroups,
					tt.wantDownstreamRequestedScopes,
					tt.wantDownstreamPKCEChallenge,
					tt.wantDownstreamPKCEChallengeMethod,
					tt.wantDownstreamNonce,
					downstreamClientID,
					tt.wantDownstreamRedirectURI,
					tt.wantDownstreamCustomSessionData,
				)
			case tt.wantRedirectToLoginPageError != "":
				// Expecting an error redirect to the login UI page.
				require.Equal(t, tt.wantBodyString, rsp.Body.String())
				expectedLocation := downstreamIssuer + oidc.PinnipedLoginPath +
					"?err=" + tt.wantRedirectToLoginPageError + "&state=" + happyEncodedUpstreamState
				require.Equal(t, expectedLocation, actualLocation)
				require.Len(t, kubeClient.Actions(), tt.wantUnnecessaryStoredRecords)
			case tt.wantRedirectLocationString != "":
				// Expecting an error redirect to the client.
				require.Equal(t, tt.wantBodyString, rsp.Body.String())
				require.Equal(t, tt.wantRedirectLocationString, actualLocation)
				require.Len(t, kubeClient.Actions(), tt.wantUnnecessaryStoredRecords)
			case tt.wantBodyFormResponseRegexp != "":
				// Expecting the body of the response to be a html page with a form (for "response_mode=form_post").
				_, hasLocationHeader := rsp.Header()["Location"]
				require.False(t, hasLocationHeader)
				oidctestutil.RequireAuthCodeRegexpMatch(
					t,
					rsp.Body.String(),
					tt.wantBodyFormResponseRegexp,
					kubeClient,
					secretsClient,
					kubeOauthStore,
					tt.wantDownstreamGrantedScopes,
					tt.wantDownstreamIDTokenSubject,
					tt.wantDownstreamIDTokenUsername,
					tt.wantDownstreamIDTokenGroups,
					tt.wantDownstreamRequestedScopes,
					tt.wantDownstreamPKCEChallenge,
					tt.wantDownstreamPKCEChallengeMethod,
					tt.wantDownstreamNonce,
					downstreamClientID,
					tt.wantDownstreamRedirectURI,
					tt.wantDownstreamCustomSessionData,
				)
			default:
				require.Failf(t, "test should have expected a redirect or form body",
					"actual location was %q", actualLocation)
			}
		})
	}
}
