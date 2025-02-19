// Copyright 2020-2022 the Pinniped contributors. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package callback provides a handler for the OIDC callback endpoint.
package callback

import (
	"net/http"
	"net/url"

	"github.com/ory/fosite"

	"go.pinniped.dev/internal/httputil/httperr"
	"go.pinniped.dev/internal/httputil/securityheader"
	"go.pinniped.dev/internal/oidc"
	"go.pinniped.dev/internal/oidc/downstreamsession"
	"go.pinniped.dev/internal/oidc/provider"
	"go.pinniped.dev/internal/oidc/provider/formposthtml"
	"go.pinniped.dev/internal/plog"
)

func NewHandler(
	upstreamIDPs oidc.UpstreamOIDCIdentityProvidersLister,
	oauthHelper fosite.OAuth2Provider,
	stateDecoder, cookieDecoder oidc.Decoder,
	redirectURI string,
) http.Handler {
	handler := httperr.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		state, err := validateRequest(r, stateDecoder, cookieDecoder)
		if err != nil {
			return err
		}

		upstreamIDPConfig := findUpstreamIDPConfig(state.UpstreamName, upstreamIDPs)
		if upstreamIDPConfig == nil {
			plog.Warning("upstream provider not found")
			return httperr.New(http.StatusUnprocessableEntity, "upstream provider not found")
		}

		downstreamAuthParams, err := url.ParseQuery(state.AuthParams)
		if err != nil {
			plog.Error("error reading state downstream auth params", err)
			return httperr.New(http.StatusBadRequest, "error reading state downstream auth params")
		}

		// Recreate enough of the original authorize request so we can pass it to NewAuthorizeRequest().
		reconstitutedAuthRequest := &http.Request{Form: downstreamAuthParams}
		authorizeRequester, err := oauthHelper.NewAuthorizeRequest(r.Context(), reconstitutedAuthRequest)
		if err != nil {
			plog.Error("error using state downstream auth params", err)
			return httperr.New(http.StatusBadRequest, "error using state downstream auth params")
		}

		// Automatically grant the openid, offline_access, and pinniped:request-audience scopes, but only if they were requested.
		downstreamsession.GrantScopesIfRequested(authorizeRequester)

		token, err := upstreamIDPConfig.ExchangeAuthcodeAndValidateTokens(
			r.Context(),
			authcode(r),
			state.PKCECode,
			state.Nonce,
			redirectURI,
		)
		if err != nil {
			plog.WarningErr("error exchanging and validating upstream tokens", err, "upstreamName", upstreamIDPConfig.GetName())
			return httperr.New(http.StatusBadGateway, "error exchanging and validating upstream tokens")
		}

		subject, username, groups, err := downstreamsession.GetDownstreamIdentityFromUpstreamIDToken(upstreamIDPConfig, token.IDToken.Claims)
		if err != nil {
			return httperr.Wrap(http.StatusUnprocessableEntity, err.Error(), err)
		}

		customSessionData, err := downstreamsession.MakeDownstreamOIDCCustomSessionData(upstreamIDPConfig, token)
		if err != nil {
			return httperr.Wrap(http.StatusUnprocessableEntity, err.Error(), err)
		}

		openIDSession := downstreamsession.MakeDownstreamSession(subject, username, groups, customSessionData)

		authorizeResponder, err := oauthHelper.NewAuthorizeResponse(r.Context(), authorizeRequester, openIDSession)
		if err != nil {
			plog.WarningErr("error while generating and saving authcode", err, "upstreamName", upstreamIDPConfig.GetName())
			return httperr.Wrap(http.StatusInternalServerError, "error while generating and saving authcode", err)
		}

		oauthHelper.WriteAuthorizeResponse(w, authorizeRequester, authorizeResponder)

		return nil
	})
	return securityheader.WrapWithCustomCSP(handler, formposthtml.ContentSecurityPolicy())
}

func authcode(r *http.Request) string {
	return r.FormValue("code")
}

func validateRequest(r *http.Request, stateDecoder, cookieDecoder oidc.Decoder) (*oidc.UpstreamStateParamData, error) {
	if r.Method != http.MethodGet {
		return nil, httperr.Newf(http.StatusMethodNotAllowed, "%s (try GET)", r.Method)
	}

	_, decodedState, err := oidc.ReadStateParamAndValidateCSRFCookie(r, cookieDecoder, stateDecoder)
	if err != nil {
		plog.InfoErr("state or CSRF error", err)
		return nil, err
	}

	if authcode(r) == "" {
		plog.Info("code param not found")
		return nil, httperr.New(http.StatusBadRequest, "code param not found")
	}

	return decodedState, nil
}

func findUpstreamIDPConfig(upstreamName string, upstreamIDPs oidc.UpstreamOIDCIdentityProvidersLister) provider.UpstreamOIDCIdentityProviderI {
	for _, p := range upstreamIDPs.GetOIDCIdentityProviders() {
		if p.GetName() == upstreamName {
			return p
		}
	}
	return nil
}
