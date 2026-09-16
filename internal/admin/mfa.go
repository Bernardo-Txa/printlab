package admin

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

const (
	MFAPendingCookieName = "printlab_admin_mfa_pending"
	MFAPendingTTL        = 10 * time.Minute
	maxPendingTokenBytes = 3800
)

func (s *Service) WritePendingCookie(w http.ResponseWriter, token string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name: MFAPendingCookieName, Value: token, Path: "/admin/mfa",
		Expires: expiresAt, MaxAge: int(MFAPendingTTL.Seconds()),
		HttpOnly: true, SameSite: http.SameSiteStrictMode, Secure: s.tokens.secure,
	})
}

func (s *Service) ClearPendingCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name: MFAPendingCookieName, Path: "/admin/mfa", Expires: time.Unix(0, 0), MaxAge: -1,
		HttpOnly: true, SameSite: http.SameSiteStrictMode, Secure: s != nil && s.tokens != nil && s.tokens.secure,
	})
}

func pendingToken(r *http.Request) string {
	cookie, err := r.Cookie(MFAPendingCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func (s *Service) pendingUser(ctx context.Context, token string) (AuthUser, error) {
	if !s.Available() {
		return AuthUser{}, ErrUnavailable
	}
	if token == "" || len(token) > maxPendingTokenBytes {
		return AuthUser{}, ErrUnauthenticated
	}
	user, err := s.auth.GetUser(ctx, token)
	if err != nil {
		return AuthUser{}, err
	}
	if normalizeUUID(user.ID) != s.adminUserID {
		return AuthUser{}, ErrUnauthenticated
	}
	// Claims are read only after this exact token has been authenticated remotely.
	claims, err := authenticatedClaims(token)
	now := s.now().UTC()
	if err != nil || claims.Subject != user.ID || claims.AAL != "aal1" || claims.IssuedAt <= 0 || time.Unix(claims.IssuedAt, 0).After(now.Add(30*time.Second)) || !time.Unix(claims.IssuedAt, 0).Add(MFAPendingTTL).After(now) || !time.Unix(claims.ExpiresAt, 0).After(now) {
		return AuthUser{}, ErrUnauthenticated
	}
	return user, nil
}

func verifiedTOTP(user AuthUser) []MFAFactor {
	var factors []MFAFactor
	for _, factor := range user.Factors {
		if ValidUUID(factor.ID) && factor.Type == "totp" && factor.Status == "verified" {
			factors = append(factors, factor)
		}
	}
	return factors
}

func (s *Service) MFAPage(ctx context.Context, r *http.Request, setup bool) (MFAPage, error) {
	user, err := s.pendingUser(ctx, pendingToken(r))
	if err != nil {
		return MFAPage{}, err
	}
	factors := verifiedTOTP(user)
	if len(factors) > 0 {
		return MFAPage{Factors: factors}, nil
	}
	page := MFAPage{Setup: true}
	if !setup {
		return page, nil
	}
	// Supabase enforces AAL2 for deleting verified factors; this AAL1 flow only removes abandoned TOTP enrollments.
	for _, factor := range user.Factors {
		if factor.Type == "totp" && factor.Status == "unverified" {
			if err := s.auth.UnenrollFactor(ctx, pendingToken(r), factor.ID); err != nil {
				return MFAPage{}, err
			}
		}
	}
	enrollment, err := s.auth.EnrollTOTP(ctx, pendingToken(r))
	if err != nil {
		return MFAPage{}, err
	}
	page.FactorID = enrollment.ID
	page.Secret = enrollment.TOTP.Secret
	page.QRCode = "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString([]byte(enrollment.TOTP.QRCode))
	return page, nil
}

func (s *Service) CompleteMFA(ctx context.Context, r *http.Request, setup bool, factorID, code string) (LoginResult, MFAPage, error) {
	user, err := s.pendingUser(ctx, pendingToken(r))
	if err != nil {
		return LoginResult{}, MFAPage{}, err
	}
	page := MFAPage{Setup: setup, Factors: verifiedTOTP(user)}
	wantedStatus := "verified"
	if setup {
		if len(page.Factors) != 0 {
			return LoginResult{}, page, ErrMFAFactor
		}
		wantedStatus = "unverified"
	}
	validFactor := false
	for _, factor := range user.Factors {
		if ValidUUID(factorID) && factor.ID == factorID && factor.Type == "totp" && factor.Status == wantedStatus {
			validFactor = true
		}
	}
	if !validFactor {
		return LoginResult{}, page, ErrMFAFactor
	}
	page.FactorID = factorID
	if !validTOTPCode(code) {
		return LoginResult{}, page, ErrMFAInvalidCode
	}
	challengeID, err := s.auth.Challenge(ctx, pendingToken(r), factorID)
	if err != nil {
		return LoginResult{}, page, err
	}
	result, err := s.auth.Verify(ctx, pendingToken(r), factorID, challengeID, code)
	if err != nil {
		return LoginResult{}, page, err
	}
	verifiedUser, err := s.auth.GetUser(ctx, result.AccessToken)
	if err != nil {
		return LoginResult{}, MFAPage{}, err
	}
	if normalizeUUID(result.User.ID) != s.adminUserID || normalizeUUID(verifiedUser.ID) != s.adminUserID {
		return LoginResult{}, MFAPage{}, ErrUnauthenticated
	}
	claims, err := authenticatedClaims(result.AccessToken)
	if err != nil || claims.Subject != verifiedUser.ID || claims.AAL != "aal2" || !time.Unix(claims.ExpiresAt, 0).After(s.now()) {
		return LoginResult{}, MFAPage{}, ErrUnauthenticated
	}
	login, err := s.createMFASession(ctx)
	return login, MFAPage{}, err
}

func validTOTPCode(code string) bool {
	if len(code) != 6 {
		return false
	}
	for _, c := range code {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

type authClaims struct {
	Subject   string `json:"sub"`
	AAL       string `json:"aal"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
}

// Not a signature verifier: callers must first authenticate the token using GetUser.
func authenticatedClaims(token string) (authClaims, error) {
	var claims authClaims
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return claims, ErrUnauthenticated
	}
	data, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || json.Unmarshal(data, &claims) != nil {
		return authClaims{}, ErrUnauthenticated
	}
	return claims, nil
}
