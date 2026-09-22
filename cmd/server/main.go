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
	"github.com/Bernardo-Txa/printlab/internal/customerauth"
	"github.com/Bernardo-Txa/printlab/internal/customers"
	"github.com/Bernardo-Txa/printlab/internal/database"
	ordersdomain "github.com/Bernardo-Txa/printlab/internal/orders"
	paymentsdomain "github.com/Bernardo-Txa/printlab/internal/payments"
	"github.com/Bernardo-Txa/printlab/internal/products"
	shippingdomain "github.com/Bernardo-Txa/printlab/internal/shipping"
	webfiles "github.com/Bernardo-Txa/printlab/web"
	"github.com/Bernardo-Txa/printlab/web/templates"
)

const (
	readyTimeout            = 3 * time.Second
	serverReadHeaderTimeout = 5 * time.Second
	serverReadTimeout       = 15 * time.Second
	serverWriteTimeout      = 30 * time.Second
	serverIdleTimeout       = 60 * time.Second
)

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

	server := newHTTPServer(addr, newHandler(db, cfg))

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}

func newHTTPServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: serverReadHeaderTimeout,
		ReadTimeout:       serverReadTimeout,
		WriteTimeout:      serverWriteTimeout,
		IdleTimeout:       serverIdleTimeout,
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
	var customerAuth customerAuthService
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
		if cfg.SupabaseURL != "" && cfg.SupabasePublishableKey != "" {
			authClient, err := customerauth.NewSupabaseClient(customerauth.SupabaseClientConfig{
				SupabaseURL:    cfg.SupabaseURL,
				PublishableKey: cfg.SupabasePublishableKey,
			})
			if err != nil {
				log.Print("customer authentication unavailable")
			} else {
				customerAuth = customerauth.NewService(
					authClient,
					customerauth.NewCookieManager(customerauth.CookieOptions{Secure: secureCartCookies(cfg)}),
					cfg.SiteURL,
				)
			}
		}
		if cfg.AdminAuthConfigured {
			authClient, err := admindomain.NewSupabaseAuthClient(admindomain.SupabaseAuthClientConfig{
				SupabaseURL:    cfg.SupabaseURL,
				PublishableKey: cfg.SupabasePublishableKey,
			})
			if err != nil {
				log.Print("admin authentication unavailable")
			} else {
				adminRepository := admindomain.NewPostgresRepository(db.Pool())
				var storageClient admindomain.StorageClient
				if cfg.SupabaseStorageConfigured {
					storageClient, err = admindomain.NewSupabaseStorageClient(admindomain.SupabaseStorageClientConfig{
						SupabaseURL: cfg.SupabaseURL,
						SecretKey:   cfg.SupabaseSecretKey,
					})
					if err != nil {
						log.Print("admin storage unavailable")
					}
				}
				adminPanel = admindomain.NewService(
					authClient,
					adminRepository,
					cfg.AdminSupabaseUserID,
					admindomain.CookieOptions{Secure: secureCartCookies(cfg)},
					admindomain.WithAdminSupabaseURL(cfg.SupabaseURL),
					admindomain.WithStorageClient(storageClient),
				)
			}
		}
	}

	return newHandlerWithServicesAndOrdersAndCustomerAuthAndSupabaseURL(db, catalog, shoppingCart, checkoutDetails, checkoutShipping, orderReview, cartdomain.NewCookieManager(cartdomain.CookieOptions{
		Secure: secureCartCookies(cfg),
	}), postalCodeLookup, payment, adminPanel, customerAuth, cfg.SiteURL, cfg.SupabaseURL)
}

func newHandlerWithCatalog(db *database.Database, catalog catalogService) http.Handler {
	return newHandlerWithServices(db, catalog, nil, nil, nil, cartdomain.NewCookieManager(cartdomain.CookieOptions{}), nil, "")
}

func newHandlerWithServices(db *database.Database, catalog catalogService, shoppingCart cartService, checkoutDetails checkoutDetailsService, checkoutShipping checkoutShippingService, cartCookies *cartdomain.CookieManager, postalCodeLookup postalCodeLookupService, siteURL string) http.Handler {
	return newHandlerWithServicesAndOrders(db, catalog, shoppingCart, checkoutDetails, checkoutShipping, nil, cartCookies, postalCodeLookup, nil, nil, siteURL)
}

func newHandlerWithServicesAndOrders(db *database.Database, catalog catalogService, shoppingCart cartService, checkoutDetails checkoutDetailsService, checkoutShipping checkoutShippingService, orderReview orderReviewService, cartCookies *cartdomain.CookieManager, postalCodeLookup postalCodeLookupService, payment paymentService, adminPanel adminPanelService, siteURL string) http.Handler {
	return newHandlerWithServicesAndOrdersAndSupabaseURL(db, catalog, shoppingCart, checkoutDetails, checkoutShipping, orderReview, cartCookies, postalCodeLookup, payment, adminPanel, siteURL, "")
}

func newHandlerWithServicesAndOrdersAndSupabaseURL(db *database.Database, catalog catalogService, shoppingCart cartService, checkoutDetails checkoutDetailsService, checkoutShipping checkoutShippingService, orderReview orderReviewService, cartCookies *cartdomain.CookieManager, postalCodeLookup postalCodeLookupService, payment paymentService, adminPanel adminPanelService, siteURL string, supabaseURL string) http.Handler {
	return newHandlerWithServicesAndOrdersAndCustomerAuthAndSupabaseURL(db, catalog, shoppingCart, checkoutDetails, checkoutShipping, orderReview, cartCookies, postalCodeLookup, payment, adminPanel, nil, siteURL, supabaseURL)
}

func newHandlerWithServicesAndOrdersAndCustomerAuthAndSupabaseURL(db *database.Database, catalog catalogService, shoppingCart cartService, checkoutDetails checkoutDetailsService, checkoutShipping checkoutShippingService, orderReview orderReviewService, cartCookies *cartdomain.CookieManager, postalCodeLookup postalCodeLookupService, payment paymentService, adminPanel adminPanelService, customerAuth customerAuthService, siteURL string, supabaseURL string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", homeHandler)
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("GET /ready", readyHandler(db))
	mux.HandleFunc("GET /robots.txt", robotsHandler(siteURL))
	mux.HandleFunc("GET /sitemap.xml", sitemapHandler(catalog, siteURL))
	mux.HandleFunc("GET /produtos", catalogHandler(catalog))
	mux.HandleFunc("GET /produtos/{slug}", productHandler(catalog))
	mux.HandleFunc("GET /cadastro", signupPageHandler(customerAuth))
	mux.HandleFunc("POST /cadastro", signupHandler(customerAuth))
	mux.HandleFunc("GET /login", loginPageHandler(customerAuth))
	mux.HandleFunc("POST /login", loginHandler(customerAuth))
	mux.HandleFunc("GET /recuperar-senha", recoveryPageHandler(customerAuth))
	mux.HandleFunc("POST /recuperar-senha", recoveryHandler(customerAuth))
	mux.HandleFunc("GET /recuperar-senha/nova", newPasswordPageHandler(customerAuth))
	mux.HandleFunc("POST /recuperar-senha/nova", newPasswordHandler(customerAuth))
	mux.HandleFunc("GET /auth/callback", authCallbackHandler(customerAuth))
	mux.HandleFunc("POST /auth/session", authSessionHandler(customerAuth))
	mux.HandleFunc("GET /conta", accountHandler(customerAuth, orderReview))
	mux.HandleFunc("POST /logout", logoutHandler(customerAuth))
	mux.HandleFunc("GET /logout", methodNotAllowedHandler(http.MethodPost))
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
	mux.HandleFunc("GET /admin/mfa/setup", adminMFAPageHandler(adminPanel, true))
	mux.HandleFunc("POST /admin/mfa/setup", adminMFAVerifyHandler(adminPanel, siteURL, true))
	mux.HandleFunc("GET /admin/mfa/challenge", adminMFAPageHandler(adminPanel, false))
	mux.HandleFunc("POST /admin/mfa/challenge", adminMFAVerifyHandler(adminPanel, siteURL, false))
	mux.HandleFunc("POST /admin/mfa/cancel", adminMFACancelHandler(adminPanel, siteURL))
	mux.HandleFunc("GET /admin", adminDashboardHandler(adminPanel))
	mux.HandleFunc("GET /admin/pedidos", adminOrdersHandler(adminPanel))
	mux.HandleFunc("GET /admin/pedidos/{orderID}", adminOrderDetailHandler(adminPanel))
	mux.HandleFunc("POST /admin/pedidos/{orderID}/producao", adminProductionStatusHandler(adminPanel, siteURL))
	mux.HandleFunc("POST /admin/pedidos/{orderID}/envio", adminShippingStatusHandler(adminPanel, siteURL))
	mux.HandleFunc("GET /admin/produtos", adminProductsHandler(adminPanel))
	mux.HandleFunc("GET /admin/produtos/novo", adminNewProductHandler(adminPanel))
	mux.HandleFunc("POST /admin/produtos", adminCreateProductHandler(adminPanel, siteURL))
	mux.HandleFunc("GET /admin/produtos/{productID}", adminProductDetailHandler(adminPanel))
	mux.HandleFunc("POST /admin/produtos/{productID}", adminUpdateProductHandler(adminPanel, siteURL))
	mux.HandleFunc("GET /admin/produtos/{productID}/imagens", adminProductImagesHandler(adminPanel))
	mux.HandleFunc("POST /admin/produtos/{productID}/imagens/upload-url", adminImageUploadURLHandler(adminPanel, siteURL))
	mux.HandleFunc("POST /admin/produtos/{productID}/imagens/finalizar", adminImageFinalizeHandler(adminPanel, siteURL))
	mux.HandleFunc("POST /admin/produtos/{productID}/imagens/{imageID}/substituir-url", adminImageReplaceURLHandler(adminPanel, siteURL))
	mux.HandleFunc("POST /admin/produtos/{productID}/imagens/{imageID}/finalizar-substituicao", adminImageReplaceFinalizeHandler(adminPanel, siteURL))
	mux.HandleFunc("POST /admin/produtos/{productID}/imagens/{imageID}/remover", adminImageRemoveHandler(adminPanel, siteURL))
	mux.HandleFunc("POST /admin/produtos/{productID}/imagens/{imageID}/ordem", adminImageOrderHandler(adminPanel, siteURL))
	mux.HandleFunc("POST /admin/produtos/{productID}/imagens/{imageID}/principal", adminImagePrimaryHandler(adminPanel, siteURL))
	mux.HandleFunc("GET /admin/produtos/{productID}/variantes/nova", adminNewVariantHandler(adminPanel))
	mux.HandleFunc("POST /admin/produtos/{productID}/variantes", adminCreateVariantHandler(adminPanel, siteURL))
	mux.HandleFunc("GET /admin/produtos/{productID}/variantes/{variantID}", adminVariantDetailHandler(adminPanel))
	mux.HandleFunc("POST /admin/produtos/{productID}/variantes/{variantID}", adminUpdateVariantHandler(adminPanel, siteURL))
	mux.HandleFunc("POST /admin/produtos/{productID}/variantes/{variantID}/receita", adminAddRecipeComponentHandler(adminPanel, siteURL))
	mux.HandleFunc("POST /admin/produtos/{productID}/variantes/{variantID}/receita/{componentID}", adminUpdateRecipeComponentHandler(adminPanel, siteURL))
	mux.HandleFunc("POST /admin/produtos/{productID}/variantes/{variantID}/receita/{componentID}/remover", adminRemoveRecipeComponentHandler(adminPanel, siteURL))
	mux.HandleFunc("GET /admin/categorias", adminCategoriesHandler(adminPanel))
	mux.HandleFunc("GET /admin/categorias/nova", adminNewCategoryHandler(adminPanel))
	mux.HandleFunc("POST /admin/categorias", adminCreateCategoryHandler(adminPanel, siteURL))
	mux.HandleFunc("GET /admin/categorias/{categoryID}", adminCategoryDetailHandler(adminPanel))
	mux.HandleFunc("POST /admin/categorias/{categoryID}", adminUpdateCategoryHandler(adminPanel, siteURL))
	mux.HandleFunc("GET /admin/materiais", adminMaterialsHandler(adminPanel))
	mux.HandleFunc("GET /admin/materiais/novo", adminNewMaterialHandler(adminPanel))
	mux.HandleFunc("POST /admin/materiais", adminCreateMaterialHandler(adminPanel, siteURL))
	mux.HandleFunc("GET /admin/materiais/{materialID}", adminMaterialDetailHandler(adminPanel))
	mux.HandleFunc("POST /admin/materiais/{materialID}", adminUpdateMaterialHandler(adminPanel, siteURL))
	mux.HandleFunc("GET /admin/cores", adminColorsHandler(adminPanel))
	mux.HandleFunc("GET /admin/cores/nova", adminNewColorHandler(adminPanel))
	mux.HandleFunc("POST /admin/cores", adminCreateColorHandler(adminPanel, siteURL))
	mux.HandleFunc("GET /admin/cores/{colorID}", adminColorDetailHandler(adminPanel))
	mux.HandleFunc("POST /admin/cores/{colorID}", adminUpdateColorHandler(adminPanel, siteURL))
	mux.HandleFunc("GET /admin/caixas", adminBoxesHandler(adminPanel))
	mux.HandleFunc("GET /admin/caixas/nova", adminNewBoxHandler(adminPanel))
	mux.HandleFunc("POST /admin/caixas", adminCreateBoxHandler(adminPanel, siteURL))
	mux.HandleFunc("GET /admin/caixas/{boxID}", adminBoxDetailHandler(adminPanel))
	mux.HandleFunc("POST /admin/caixas/{boxID}", adminUpdateBoxHandler(adminPanel, siteURL))
	mux.HandleFunc("POST /admin/logout", adminLogoutHandler(adminPanel, siteURL))
	mux.HandleFunc("GET /admin/logout", methodNotAllowedHandler(http.MethodPost))
	mux.HandleFunc("GET /admin/{path...}", adminProtectedNotFoundHandler(adminPanel))
	mux.HandleFunc("POST /admin/{path...}", adminProtectedNotFoundHandler(adminPanel))
	mux.Handle("GET /static/", staticFileHandler(webfiles.StaticFS()))

	return securityMiddleware(customerSessionMiddleware(mux, customerAuth), siteURL, supabaseURL)
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

		w.Header().Set("Cache-Control", staticCacheControl(name))

		request := r.Clone(r.Context())
		request.URL.Path = "/static/" + name
		fileServer.ServeHTTP(w, request)
	})
}

func staticCacheControl(name string) string {
	if strings.Contains(path.Base(name), "-v1.") {
		return "public, max-age=31536000, immutable"
	}

	return "public, max-age=3600"
}
