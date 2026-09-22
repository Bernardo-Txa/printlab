package components

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/Bernardo-Txa/printlab/internal/customerauth"
)

func TestHeaderRendersBrandNavigationAndGuestActions(t *testing.T) {
	html := renderComponent(t, Header())

	for _, expected := range []string{
		`src="/static/images/branding/logo-printlab-small-v1.webp"`,
		`href="/#inicio">Início`,
		`href="/produtos">Produtos`,
		`href="/#como-funciona">Como funciona`,
		`href="/#lab">Sobre`,
		`href="/#contato">Contato`,
		`href="/carrinho"`,
		`href="/login"`,
		`Entrar`,
		`Carrinho`,
	} {
		if !strings.Contains(html, expected) {
			t.Fatalf("expected guest header to contain %q, got %s", expected, html)
		}
	}
	for _, forbidden := range []string{`<input`, `type="search"`, `site-header-cart-badge`, `>0</`} {
		if strings.Contains(html, forbidden) {
			t.Fatalf("expected header not to render fake search or cart badge %q, got %s", forbidden, html)
		}
	}
}

func TestHeaderRendersAuthenticatedActionsAndPostLogout(t *testing.T) {
	ctx := customerauth.WithProfile(context.Background(), customerauth.Profile{ID: "11111111-1111-1111-1111-111111111111", Name: "Cliente", Email: "cliente@example.com"})
	html := renderHeaderWithContext(t, ctx)

	for _, expected := range []string{
		`href="/conta"`,
		`Minha conta`,
		`method="post" action="/logout"`,
		`type="submit">Sair`,
		`href="/carrinho"`,
	} {
		if !strings.Contains(html, expected) {
			t.Fatalf("expected authenticated header to contain %q, got %s", expected, html)
		}
	}
	if strings.Contains(html, `href="/login"`) || strings.Contains(html, `>Entrar<`) {
		t.Fatalf("expected authenticated header not to show login action, got %s", html)
	}
}

func renderHeaderWithContext(t *testing.T, ctx context.Context) string {
	t.Helper()

	var buffer bytes.Buffer
	if err := Header().Render(ctx, &buffer); err != nil {
		t.Fatalf("expected header render to succeed, got %v", err)
	}
	return buffer.String()
}
