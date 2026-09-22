package components

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/Bernardo-Txa/printlab/internal/customerauth"
)

func TestHeaderRendersBrandNavigationAndGuestActions(t *testing.T) {
	html := renderComponent(t, Header())

	for _, expected := range []string{
		`src="/static/images/branding/logo-printlab-small-v1.webp"`,
		`<nav class="site-header-actions" aria-label="Ações do usuário">`,
		`href="/#inicio">Início`,
		`href="/produtos">Produtos`,
		`href="/#como-funciona">Como funciona`,
		`href="/#lab">Sobre`,
		`href="/#contato">Contato`,
		`href="/carrinho"`,
		`href="/login"`,
		`aria-label="Entrar"`,
		`aria-label="Carrinho"`,
		`class="sr-only">Entrar`,
		`class="sr-only">Carrinho`,
	} {
		if !strings.Contains(html, expected) {
			t.Fatalf("expected guest header to contain %q, got %s", expected, html)
		}
	}
	for _, forbidden := range []string{`<input`, `type="search"`, `site-header-cart-badge`, `>0</`, `badge-count`} {
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
		`aria-label="Minha conta"`,
		`aria-label="Carrinho"`,
		`class="sr-only">Minha conta`,
	} {
		if !strings.Contains(html, expected) {
			t.Fatalf("expected authenticated header to contain %q, got %s", expected, html)
		}
	}
	if strings.Contains(html, `href="/login"`) || strings.Contains(html, `>Entrar<`) {
		t.Fatalf("expected authenticated header not to show login action, got %s", html)
	}
}

func TestFooterKeepsReadableLinksAndHeaderColorsScoped(t *testing.T) {
	html := renderComponent(t, Footer())
	if !strings.Contains(html, `aria-label="Navegação do rodapé"`) || !strings.Contains(html, `class="nav-link"`) {
		t.Fatalf("expected footer navigation links, got %s", html)
	}

	css, err := os.ReadFile("../../web/assets/css/app.css")
	if err != nil {
		t.Fatalf("expected source CSS to be readable, got %v", err)
	}
	cssText := string(css)
	for _, expected := range []string{".site-header .nav-link:focus-visible", "footer .nav-link:focus-visible"} {
		if !strings.Contains(cssText, expected) {
			t.Fatalf("expected CSS scoping assertion %q", expected)
		}
	}
	if strings.Contains(cssText, "\n  .nav-link:focus-visible {") {
		t.Fatal("expected dark header focus styles not to apply globally")
	}
	if !strings.Contains(cssText, ".site-header .nav-link-cart::before") {
		t.Fatal("expected cart indicator to remain scoped to the header")
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
