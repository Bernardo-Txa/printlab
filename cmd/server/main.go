package main

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"path"
	"strings"
	"time"

	admindomain "github.com/Bernardo-Txa/printlab/internal/admin"
	cartdomain "github.com/Bernardo-Txa/printlab/internal/cart"
	"github.com/Bernardo-Txa/printlab/internal/config"
	"github.com/Bernardo-Txa/printlab/internal/customers"
	"github.com/Bernardo-Txa/printlab/internal/database"
	ordersdomain "github.com/Bernardo-Txa/printlab/internal/orders"
	paymentsdomain "github.com/Bernardo-Txa/printlab/internal/payments"
	"github.com/Bernardo-Txa/printlab/internal/products"
	shippingdomain "github.com/Bernardo-Txa/printlab/internal/shipping"
	webfiles "github.com/Bernardo-Txa/printlab/web"
	"github.com/Bernardo-Txa/printlab/web/templates"
)

const readyTimeout = 3 * time.Second

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("application configuration error")
	}

	db, err := database.New(context.Background(), database.Config{
		DatabaseURL: cfg.DatabaseURL,
		MaxConns:    cfg.DBMaxConns,
	})
	if err != nil {
		log.Fatal("database configuration error")
	}
	defer db.Close()

	if db.Configured() {
		log.Print("database configured")
	}

	addr := ":" + cfg.Port
	log.Printf("printlab web listening on %s", addr)

	if err := http.ListenAndServe(addr, newHandler(db, cfg)); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}

func newHandler(db *database.Database, cfg config.Config) http.Handler {
	var catalog catalogService
	var shoppingCart cartService
	var checkoutDetails checkoutDetailsService
	var checkoutShipping checkoutShippingService
	var orderReview orderReviewService
	var payment paymentService
	var adminPanel adminPanelService
	postalCodeLookup := customers.NewViaCEPClient()
	if db != nil && db.Configured() {
		supabaseURL := cfg.SupabaseURL
		customerRepository := customers.NewPostgresRepository(db.Pool())
		catalog = products.NewService(
			products.NewPostgresRepository(db.Pool()),
			products.WithSupabaseURL(supabaseURL),
		)
		shoppingCart = cartdomain.NewService(
			cartdomain.NewPostgresRepository(db.Pool()),
			cartdomain.WithSupabaseURL(supabaseURL),
		)
		checkoutDetails = customers.NewService(
			customerRepository,
			shoppingCart,
		)
		var calculator shippingdomain.Calculator
		if cfg.SuperFreteConfigured {
			client, err := shippingdomain.NewSuperFreteClient(shippingdomain.SuperFreteClientConfig{
				Environment:  cfg.SuperFreteEnv,
				APIToken:     cfg.SuperFreteAPIToken,
				ContactEmail: cfg.SuperFreteContactEmail,
			})
			if err != nil {
				log.Fatal("shipping configuration error")
			}
			calculator = client
		}
		checkoutShipping = shippingdomain.NewService(
			shippingdomain.NewPostgresRepository(db.Pool()),
			shoppingCart,
			customerRepository,
			calculator,
			cfg.SuperFreteOriginPostalCode,
			cfg.SuperFreteServiceCodes,
		)
		orderReview = ordersdomain.NewService(
			ordersdomain.NewPostgresRepository(db.Pool()),
			cfg.SuperFreteOriginPostalCode,
			cfg.SuperFreteServiceCodes,
		)
		payment = paymentsdomain.NewService(
			paymentsdomain.NewPostgresRepository(db.Pool()),
			paymentsdomain.NewInfinitePayClient(),
			cfg.InfinitePayHandle,
			cfg.SiteURL,
		)
		if cfg.AdminAuthConfigured {
			authClient, err := admindomain.NewSupabaseAuthClient(admindomain.SupabaseAuthClientConfig{
				SupabaseURL:    cfg.SupabaseURL,
				PublishableKey: cfg.SupabasePublishableKey,
			})
			if err != nil {
				log.Print("admin authentication unavailable")
			} else {
				adminPanel = admindomain.NewService(
					authClient,
					admindomain.NewPostgresRepository(db.Pool()),
					cfg.AdminSupabaseUserID,
					admindomain.CookieOptions{Secure: secureCartCookies(cfg)},
				)
			}
		}
	}

	return newHandlerWithServicesAndOrders(db, catalog, shoppingCart, checkoutDetails, checkoutShipping, orderReview, cartdomain.NewCookieManager(cartdomain.CookieOptions{
		Secure: secureCartCookies(cfg),
	}), postalCodeLookup, payment, adminPanel, cfg.SiteURL)
}

func newHandlerWithCatalog(db *database.Database, catalog catalogService) http.Handler {
	return newHandlerWithServices(db, catalog, nil, nil, nil, cartdomain.NewCookieManager(cartdomain.CookieOptions{}), nil, "")
}

func newHandlerWithServices(db *database.Database, catalog catalogService, shoppingCart cartService, checkoutDetails checkoutDetailsService, checkoutShipping checkoutShippingService, cartCookies *cartdomain.CookieManager, postalCodeLookup postalCodeLookupService, siteURL string) http.Handler {
	return newHandlerWithServicesAndOrders(db, catalog, shoppingCart, checkoutDetails, checkoutShipping, nil, cartCookies, postalCodeLookup, nil, nil, siteURL)
}

func newHandlerWithServicesAndOrders(db *database.Database, catalog catalogService, shoppingCart cartService, checkoutDetails checkoutDetailsService, checkoutShipping checkoutShippingService, orderReview orderReviewService, cartCookies *cartdomain.CookieManager, postalCodeLookup postalCodeLookupService, payment paymentService, adminPanel adminPanelService, siteURL string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", homeHandler)
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("GET /ready", readyHandler(db))
	mux.HandleFunc("GET /produtos", catalogHandler(catalog))
	mux.HandleFunc("GET /produtos/{slug}", productHandler(catalog))
	mux.HandleFunc("GET /carrinho", cartPageHandler(shoppingCart, cartCookies))
	mux.HandleFunc("POST /carrinho/adicionar", addCartItemHandler(shoppingCart, cartCookies, siteURL))
	mux.HandleFunc("POST /carrinho/itens/{id}/quantidade", updateCartItemQuantityHandler(shoppingCart, cartCookies, siteURL))
	mux.HandleFunc("POST /carrinho/itens/{id}/remover", removeCartItemHandler(shoppingCart, cartCookies, siteURL))
	mux.HandleFunc("GET /checkout/dados", checkoutDetailsPageHandler(checkoutDetails, cartCookies))
	mux.HandleFunc("POST /checkout/dados", saveCheckoutDetailsHandler(checkoutDetails, cartCookies, siteURL))
	mux.HandleFunc("GET /api/cep/{cep}", postalCodeLookupHandler(postalCodeLookup))
	mux.HandleFunc("GET /checkout/frete", checkoutShippingPageHandler(checkoutShipping, cartCookies))
	mux.HandleFunc("POST /checkout/frete", selectShippingHandler(checkoutShipping, cartCookies, siteURL))
	mux.HandleFunc("GET /checkout/revisao", checkoutReviewPageHandler(orderReview, cartCookies))
	mux.HandleFunc("POST /checkout/revisao", confirmOrderHandler(orderReview, cartCookies, siteURL))
	mux.HandleFunc("GET /pedido/{id}", orderPageHandler(orderReview, payment))
	mux.HandleFunc("POST /pedido/{id}/pagar", startPaymentHandler(payment, siteURL))
	mux.HandleFunc("GET /acompanhar/{tracking_id}", orderTrackingPageHandler(orderReview))
	mux.HandleFunc("POST /acompanhar/{tracking_id}", methodNotAllowedHandler(http.MethodGet))
	mux.HandleFunc("GET /pagamento/retorno", paymentReturnHandler(payment))
	mux.HandleFunc("POST /webhooks/infinitepay", infinitePayWebhookHandler(payment))
	mux.HandleFunc("GET /webhooks/infinitepay", methodNotAllowedHandler(http.MethodPost))
	mux.HandleFunc("GET /admin/login", adminLoginPageHandler(adminPanel))
	mux.HandleFunc("POST /admin/login", adminLoginHandler(adminPanel, siteURL))
	mux.HandleFunc("GET /admin", adminDashboardHandler(adminPanel))
	mux.HandleFunc("GET /admin/pedidos", adminOrdersHandler(adminPanel))
	mux.HandleFunc("GET /admin/pedidos/{orderID}", adminOrderDetailHandler(adminPanel))
	mux.HandleFunc("POST /admin/pedidos/{orderID}/producao", adminProductionStatusHandler(adminPanel, siteURL))
	mux.HandleFunc("POST /admin/pedidos/{orderID}/envio", adminShippingStatusHandler(adminPanel, siteURL))
	mux.HandleFunc("POST /admin/logout", adminLogoutHandler(adminPanel, siteURL))
	mux.HandleFunc("GET /admin/logout", methodNotAllowedHandler(http.MethodPost))
	mux.HandleFunc("GET /admin/{path...}", adminProtectedNotFoundHandler(adminPanel))
	mux.HandleFunc("POST /admin/{path...}", adminProtectedNotFoundHandler(adminPanel))
	mux.Handle("GET /static/", staticFileHandler(webfiles.StaticFS()))

	return mux
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.Home().Render(r.Context(), w); err != nil {
		log.Printf("render home: %v", err)
	}
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintln(w, "ok")
}

func readyHandler(db *database.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")

		if db == nil || !db.Configured() {
			http.Error(w, "service unavailable", http.StatusServiceUnavailable)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), readyTimeout)
		defer cancel()

		if err := db.Ping(ctx); err != nil {
			log.Print("database unavailable")
			http.Error(w, "service unavailable", http.StatusServiceUnavailable)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintln(w, "ok")
	}
}

func staticFileHandler(staticFS fs.FS) http.Handler {
	fileServer := http.StripPrefix("/static/", http.FileServer(http.FS(staticFS)))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rel := strings.TrimPrefix(r.URL.Path, "/static/")
		clean := path.Clean("/" + rel)
		name := strings.TrimPrefix(clean, "/")

		if rel == "" || strings.HasSuffix(rel, "/") || strings.Contains(clean, "/.") {
			http.NotFound(w, r)
			return
		}

		info, err := fs.Stat(staticFS, name)
		if err != nil || info.IsDir() {
			http.NotFound(w, r)
			return
		}

		request := r.Clone(r.Context())
		request.URL.Path = "/static/" + name
		fileServer.ServeHTTP(w, request)
	})
}
