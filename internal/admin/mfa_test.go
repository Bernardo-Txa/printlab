package admin

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const testFactorID = "33333333-3333-3333-3333-333333333333"

func testAuthJWT(userID, aal string, now time.Time) string {
	data, _ := json.Marshal(authClaims{Subject: userID, AAL: aal, IssuedAt: now.Unix(), ExpiresAt: now.Add(time.Hour).Unix()})
	return "test." + base64.RawURLEncoding.EncodeToString(data) + ".signature"
}

func (c fakeAuthClient) GetUser(context.Context, string) (AuthUser, error) {
	return AuthUser{ID: c.userID, Factors: []MFAFactor{{ID: testFactorID, Type: "totp", Status: "verified"}}}, c.err
}
func (c fakeAuthClient) EnrollTOTP(context.Context, string) (TOTPEnrollment, error) {
	return TOTPEnrollment{}, c.err
}
func (c fakeAuthClient) UnenrollFactor(context.Context, string, string) error { return c.err }
func (c fakeAuthClient) Challenge(context.Context, string, string) (string, error) {
	return testFactorID, c.err
}
func (c fakeAuthClient) Verify(context.Context, string, string, string, string) (AuthSession, error) {
	return AuthSession{User: AuthUser{ID: c.userID}, AccessToken: testAuthJWT(c.userID, "aal2", c.now)}, c.err
}

func completeTestMFA(service *Service) (LoginResult, error) {
	pending, err := service.Login(context.Background(), "admin@example.com", "correct-password")
	if err != nil {
		return LoginResult{}, err
	}
	req := httptest.NewRequest(http.MethodPost, "/admin/mfa/challenge", nil)
	req.AddCookie(&http.Cookie{Name: MFAPendingCookieName, Value: pending.PendingToken})
	result, _, err := service.CompleteMFA(context.Background(), req, false, testFactorID, "123456")
	return result, err
}

func TestPasswordOnlyNeverCreatesSession(t *testing.T) {
	repo := newFakeRepository()
	service := newTestService(repo, testAdminUserID, time.Now())
	result, err := service.Login(context.Background(), "admin@example.com", "password")
	if err != nil || len(repo.sessions) != 0 || result.Token != "" || result.PendingToken == "" || result.NextPath != "/admin/mfa/challenge" {
		t.Fatal("password must produce only MFA pending state")
	}
}

func TestLegacySessionWithoutMFATimestampRejected(t *testing.T) {
	now := time.Now()
	repo := newFakeRepository()
	service := newTestService(repo, testAdminUserID, now)
	token, hash := mustTokenAndHash(t, service.tokens)
	repo.sessions[string(hash)] = Session{AuthUserID: testAdminUserID, ExpiresAt: now.Add(SessionTTL)}
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.AddCookie(service.tokens.Cookie(token, now.Add(SessionTTL)))
	if _, err := service.ResolveSession(context.Background(), req); !errors.Is(err, ErrUnauthenticated) {
		t.Fatal("legacy session must not authorize")
	}
}

func TestPendingTokenAgeEnforcedServerSide(t *testing.T) {
	now := time.Now()
	service := newTestService(newFakeRepository(), testAdminUserID, now)
	for _, issued := range []time.Time{now.Add(-MFAPendingTTL), now.Add(time.Minute)} {
		_, err := service.pendingUser(context.Background(), testAuthJWT(testAdminUserID, "aal1", issued))
		if !errors.Is(err, ErrUnauthenticated) {
			t.Fatal("expired or future pending token accepted")
		}
	}
}

func TestPendingCookieAttributes(t *testing.T) {
	for _, secure := range []bool{false, true} {
		service := NewService(fakeAuthClient{}, newFakeRepository(), testAdminUserID, CookieOptions{Secure: secure})
		rec := httptest.NewRecorder()
		expires := time.Now().UTC().Truncate(time.Second).Add(MFAPendingTTL)
		service.WritePendingCookie(rec, "token", expires)
		c := rec.Result().Cookies()[0]
		if c.Name != MFAPendingCookieName || !c.HttpOnly || c.SameSite != http.SameSiteStrictMode || c.Secure != secure || c.Path != "/admin/mfa" || c.Domain != "" || c.MaxAge != 600 || !c.Expires.Equal(expires) {
			t.Fatal("incorrect pending cookie attributes")
		}
	}
}
