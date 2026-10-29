package components

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/Bernardo-Txa/printlab/internal/cart"
	"github.com/Bernardo-Txa/printlab/internal/customerauth"
	"github.com/Bernardo-Txa/printlab/internal/seo"
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

func TestHeaderRendersRealCartCountFromContext(t *testing.T) {
	ctx := cart.WithUnitCount(context.Background(), 5)
	html := renderHeaderWithContext(t, ctx)

	for _, expected := range []string{
		`aria-label="Carrinho, 5 unidades"`,
		`class="site-header-cart-count"`,
		`>5</span>`,
	} {
		if !strings.Contains(html, expected) {
			t.Fatalf("expected counted cart header to contain %q, got %s", expected, html)
		}
	}
}

func TestHeaderCapsVisualCartCountButKeepsRealAriaCount(t *testing.T) {
	ctx := cart.WithUnitCount(context.Background(), 125)
	html := renderHeaderWithContext(t, ctx)
	if !strings.Contains(html, `aria-label="Carrinho, 125 unidades"`) || !strings.Contains(html, `>99+</span>`) {
		t.Fatalf("expected visual 99+ and real aria count, got %s", html)
	}
}

func TestHeaderOmitsCartBadgeWhenEmpty(t *testing.T) {
	html := renderComponent(t, Header())
	if strings.Contains(html, `site-header-cart-count`) || strings.Contains(html, `Carrinho, 0`) {
		t.Fatalf("expected empty cart not to render a badge, got %s", html)
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

func TestHeaderRendersServerSideCurrentPage(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		profile  bool
		active   string
		inactive []string
	}{
		{name: "products", path: "/produtos/vaso", active: `href="/produtos" aria-current="page"`, inactive: []string{`href="/carrinho" aria-current="page"`, `href="/conta" aria-current="page"`}},
		{name: "cart", path: "/carrinho", active: `href="/carrinho" aria-label="Carrinho" aria-current="page"`, inactive: []string{`href="/produtos" aria-current="page"`, `href="/conta" aria-current="page"`}},
		{name: "account", path: "/conta/pedidos", profile: true, active: `href="/conta" aria-label="Minha conta" aria-current="page"`, inactive: []string{`href="/produtos" aria-current="page"`, `href="/carrinho" aria-current="page"`}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := seo.WithRequestPath(context.Background(), tt.path)
			if tt.profile {
				ctx = customerauth.WithProfile(ctx, customerauth.Profile{ID: "11111111-1111-1111-1111-111111111111"})
			}
			html := renderHeaderWithContext(t, ctx)
			if !strings.Contains(html, tt.active) {
				t.Fatalf("expected current page marker %q, got %s", tt.active, html)
			}
			if count := strings.Count(html, `aria-current="page"`); count != 1 {
				t.Fatalf("expected exactly one current page marker, got %d in %s", count, html)
			}
			for _, forbidden := range tt.inactive {
				if strings.Contains(html, forbidden) {
					t.Fatalf("expected inactive link not to contain %q, got %s", forbidden, html)
				}
			}
		})
	}
}

func TestHeaderDoesNotMarkUnrelatedPageCurrent(t *testing.T) {
	ctx := seo.WithRequestPath(context.Background(), "/checkout/dados")
	html := renderHeaderWithContext(t, ctx)
	if strings.Contains(html, `aria-current="page"`) || strings.Contains(html, `is-active`) {
		t.Fatalf("expected unrelated page without current header item, got %s", html)
	}
}

func TestActiveNavigationScriptPreservesServerRenderedState(t *testing.T) {
	script, err := os.ReadFile("../static/js/motion.js")
	if err != nil {
		t.Fatalf("expected navigation script to be readable, got %v", err)
	}
	text := string(script)
	for _, expected := range []string{`link.classList.add("is-active")`, `link.setAttribute("aria-current"`} {
		if !strings.Contains(text, expected) {
			t.Fatalf("expected progressive navigation enhancement %q", expected)
		}
	}
	if strings.Contains(text, `removeAttribute("aria-current")`) || strings.Contains(text, `classList.remove("is-active")`) {
		t.Fatal("expected navigation enhancement not to remove server-rendered active state")
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
	if strings.Contains(cssText, ".site-header .nav-link-cart::before") {
		t.Fatal("expected cart indicator not to be invented by pseudo-element CSS")
	}
	if !strings.Contains(cssText, ".site-header-cart-count") || !strings.Contains(cssText, "bg-highlight-blue") {
		t.Fatal("expected cart count badge to remain a scoped blue header element")
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
