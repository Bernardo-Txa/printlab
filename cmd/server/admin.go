package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	admindomain "github.com/Bernardo-Txa/printlab/internal/admin"
	"github.com/Bernardo-Txa/printlab/web/templates"
)

const adminMaxFormBodyBytes = 256 << 10

type adminPanelService interface {
	Available() bool
	Login(ctx context.Context, email string, password string) (admindomain.LoginResult, error)
	ResolveSession(ctx context.Context, r *http.Request) (admindomain.Session, error)
	Logout(ctx context.Context, r *http.Request) error
	Dashboard(ctx context.Context) (admindomain.Dashboard, error)
	ListOrders(ctx context.Context, filter admindomain.OrderListFilter) (admindomain.OrderListPage, error)
	GetOrder(ctx context.Context, orderID string) (admindomain.OrderDetail, error)
	ChangeProductionStatus(ctx context.Context, orderID string, targetStatus string, actorAuthUserID string) error
	ChangeShippingStatus(ctx context.Context, orderID string, targetStatus string, actorAuthUserID string) error
	ListAdminProducts(ctx context.Context, filter admindomain.AdminProductListFilter) (admindomain.AdminProductListPage, error)
	NewAdminProduct(ctx context.Context) (admindomain.AdminProductFormPage, error)
	GetAdminProduct(ctx context.Context, productID string) (admindomain.AdminProductFormPage, error)
	CreateAdminProduct(ctx context.Context, form admindomain.AdminProductForm) (string, admindomain.AdminProductFormPage, error)
	UpdateAdminProduct(ctx context.Context, productID string, form admindomain.AdminProductForm) (admindomain.AdminProductFormPage, error)
	ListAdminCategories(ctx context.Context) (admindomain.AdminCategoryListPage, error)
	NewAdminCategory() admindomain.AdminCategoryFormPage
	GetAdminCategory(ctx context.Context, categoryID string) (admindomain.AdminCategoryFormPage, error)
	CreateAdminCategory(ctx context.Context, form admindomain.AdminCategoryForm) (string, admindomain.AdminCategoryFormPage, error)
	UpdateAdminCategory(ctx context.Context, categoryID string, form admindomain.AdminCategoryForm) (admindomain.AdminCategoryFormPage, error)
	NewAdminVariant(ctx context.Context, productID string) (admindomain.AdminVariantFormPage, error)
	GetAdminVariant(ctx context.Context, productID string, variantID string) (admindomain.AdminVariantFormPage, error)
	CreateAdminVariant(ctx context.Context, productID string, form admindomain.AdminVariantForm) (string, admindomain.AdminVariantFormPage, error)
	UpdateAdminVariant(ctx context.Context, productID string, variantID string, form admindomain.AdminVariantForm) (admindomain.AdminVariantFormPage, error)
	AddAdminRecipeComponent(ctx context.Context, productID string, variantID string, form admindomain.AdminRecipeForm) error
	UpdateAdminRecipeComponent(ctx context.Context, productID string, variantID string, componentID string, form admindomain.AdminRecipeForm) error
	RemoveAdminRecipeComponent(ctx context.Context, productID string, variantID string, componentID string) error
	ListAdminMaterials(ctx context.Context) (admindomain.AdminMaterialListPage, error)
	NewAdminMaterial() admindomain.AdminMaterialFormPage
	GetAdminMaterial(ctx context.Context, materialID string) (admindomain.AdminMaterialFormPage, error)
	CreateAdminMaterial(ctx context.Context, form admindomain.AdminMaterialForm) (string, admindomain.AdminMaterialFormPage, error)
	UpdateAdminMaterial(ctx context.Context, materialID string, form admindomain.AdminMaterialForm) (admindomain.AdminMaterialFormPage, error)
	ListAdminColors(ctx context.Context) (admindomain.AdminColorListPage, error)
	NewAdminColor() admindomain.AdminColorFormPage
	GetAdminColor(ctx context.Context, colorID string) (admindomain.AdminColorFormPage, error)
	CreateAdminColor(ctx context.Context, form admindomain.AdminColorForm) (string, admindomain.AdminColorFormPage, error)
	UpdateAdminColor(ctx context.Context, colorID string, form admindomain.AdminColorForm) (admindomain.AdminColorFormPage, error)
	ListAdminBoxes(ctx context.Context) (admindomain.AdminBoxListPage, error)
	NewAdminBox() admindomain.AdminBoxFormPage
	GetAdminBox(ctx context.Context, boxID string) (admindomain.AdminBoxFormPage, error)
	CreateAdminBox(ctx context.Context, form admindomain.AdminBoxForm) (string, admindomain.AdminBoxFormPage, error)
	UpdateAdminBox(ctx context.Context, boxID string, form admindomain.AdminBoxForm) (admindomain.AdminBoxFormPage, error)
	WriteCookie(w http.ResponseWriter, token string, expiresAt time.Time)
	ClearCookie(w http.ResponseWriter)
}

func adminLoginPageHandler(service adminPanelService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setAdminPrivateHeaders(w)
		if !adminAvailable(service) {
			renderHTML(w, r, http.StatusServiceUnavailable, templates.AdminUnavailable())
			return
		}

		renderHTML(w, r, http.StatusOK, templates.AdminLogin(""))
	}
}

func adminLoginHandler(service adminPanelService, siteURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setAdminPrivateHeaders(w)
		if !validMutationSource(r, siteURL) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if !adminAvailable(service) {
			renderHTML(w, r, http.StatusServiceUnavailable, templates.AdminUnavailable())
			return
		}
		if err := parseAdminForm(w, r); err != nil {
			http.Error(w, "invalid admin request", http.StatusBadRequest)
			return
		}

		result, err := service.Login(r.Context(), strings.TrimSpace(r.PostFormValue("email")), r.PostFormValue("password"))
		if err != nil {
			log.Print("admin login failed")
			renderHTML(w, r, http.StatusUnauthorized, templates.AdminLogin(admindomain.InvalidCredentialsMessage))
			return
		}

		log.Print("admin login succeeded")
		service.WriteCookie(w, result.Token, result.ExpiresAt)
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
	}
}

func adminDashboardHandler(service adminPanelService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setAdminPrivateHeaders(w)
		if _, ok := requireAdminSession(w, r, service); !ok {
			return
		}

		dashboard, err := service.Dashboard(r.Context())
		if err != nil {
			log.Print("admin repository error")
			renderHTML(w, r, http.StatusServiceUnavailable, templates.AdminUnavailable())
			return
		}

		renderHTML(w, r, http.StatusOK, templates.AdminDashboard(dashboard))
	}
}

func adminOrdersHandler(service adminPanelService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setAdminPrivateHeaders(w)
		if _, ok := requireAdminSession(w, r, service); !ok {
			return
		}

		page, err := service.ListOrders(r.Context(), admindomain.OrderListFilter{
			Status: r.URL.Query().Get("status"),
			Page:   adminPageQuery(r),
		})
		if err != nil {
			log.Print("admin repository error")
			renderHTML(w, r, http.StatusServiceUnavailable, templates.AdminUnavailable())
			return
		}

		renderHTML(w, r, http.StatusOK, templates.AdminOrders(page))
	}
}

func adminOrderDetailHandler(service adminPanelService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setAdminPrivateHeaders(w)
		if _, ok := requireAdminSession(w, r, service); !ok {
			return
		}

		detail, err := service.GetOrder(r.Context(), r.PathValue("orderID"))
		if err != nil {
			if errors.Is(err, admindomain.ErrInvalidOrderID) || errors.Is(err, admindomain.ErrOrderNotFound) {
				http.NotFound(w, r)
				return
			}
			log.Print("admin repository error")
			renderHTML(w, r, http.StatusServiceUnavailable, templates.AdminUnavailable())
			return
		}

		successMessage, errorMessage := adminOrderMessages(r)
		renderHTML(w, r, http.StatusOK, templates.AdminOrderDetail(detail, successMessage, errorMessage))
	}
}

func adminProductionStatusHandler(service adminPanelService, siteURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setAdminPrivateHeaders(w)
		if !validMutationSource(r, siteURL) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		session, ok := requireAdminSession(w, r, service)
		if !ok {
			return
		}
		if err := parseAdminForm(w, r); err != nil {
			http.Error(w, "invalid admin request", http.StatusBadRequest)
			return
		}

		orderID := r.PathValue("orderID")
		err := service.ChangeProductionStatus(r.Context(), orderID, r.PostFormValue("production_status"), session.AuthUserID)
		handleAdminOrderMutationResult(w, r, orderID, "producao", err)
	}
}

func adminShippingStatusHandler(service adminPanelService, siteURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setAdminPrivateHeaders(w)
		if !validMutationSource(r, siteURL) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		session, ok := requireAdminSession(w, r, service)
		if !ok {
			return
		}
		if err := parseAdminForm(w, r); err != nil {
			http.Error(w, "invalid admin request", http.StatusBadRequest)
			return
		}

		orderID := r.PathValue("orderID")
		err := service.ChangeShippingStatus(r.Context(), orderID, r.PostFormValue("shipping_status"), session.AuthUserID)
		handleAdminOrderMutationResult(w, r, orderID, "envio", err)
	}
}

func adminLogoutHandler(service adminPanelService, siteURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setAdminPrivateHeaders(w)
		r.Body = http.MaxBytesReader(w, r.Body, adminMaxFormBodyBytes)
		if !validMutationSource(r, siteURL) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if adminAvailable(service) {
			if err := service.Logout(r.Context(), r); err != nil && !errors.Is(err, admindomain.ErrUnauthenticated) {
				log.Print("admin repository error")
			}
			service.ClearCookie(w)
		}

		log.Print("admin logout")
		http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
	}
}

func adminProtectedNotFoundHandler(service adminPanelService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setAdminPrivateHeaders(w)
		if _, ok := requireAdminSession(w, r, service); !ok {
			return
		}

		http.NotFound(w, r)
	}
}

func requireAdminSession(w http.ResponseWriter, r *http.Request, service adminPanelService) (admindomain.Session, bool) {
	if !adminAvailable(service) {
		http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
		return admindomain.Session{}, false
	}

	session, err := service.ResolveSession(r.Context(), r)
	if err != nil {
		if errors.Is(err, admindomain.ErrSessionExpired) {
			log.Print("admin session expired")
		}
		service.ClearCookie(w)
		http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
		return admindomain.Session{}, false
	}

	return session, true
}

func adminAvailable(service adminPanelService) bool {
	return service != nil && service.Available()
}

func setAdminPrivateHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", checkoutPrivateCacheControl)
	w.Header().Set("X-Robots-Tag", "noindex, nofollow, noarchive")
	w.Header().Set("Referrer-Policy", "same-origin")
}

func parseAdminForm(w http.ResponseWriter, r *http.Request) error {
	r.Body = http.MaxBytesReader(w, r.Body, adminMaxFormBodyBytes)
	return r.ParseForm()
}

func adminPageQuery(r *http.Request) int {
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		return 1
	}

	return page
}

func handleAdminOrderMutationResult(w http.ResponseWriter, r *http.Request, orderID string, operation string, err error) {
	if err == nil {
		http.Redirect(w, r, adminOrderDetailPath(orderID)+"?ok="+operation, http.StatusSeeOther)
		return
	}
	if errors.Is(err, admindomain.ErrInvalidOrderID) || errors.Is(err, admindomain.ErrOrderNotFound) {
		http.NotFound(w, r)
		return
	}
	if errors.Is(err, admindomain.ErrInvalidTransition) || errors.Is(err, admindomain.ErrTransitionConflict) {
		http.Redirect(w, r, adminOrderDetailPath(orderID)+"?erro=transicao", http.StatusSeeOther)
		return
	}

	log.Print("admin repository error")
	renderHTML(w, r, http.StatusServiceUnavailable, templates.AdminUnavailable())
}

func adminOrderDetailPath(orderID string) string {
	return "/admin/pedidos/" + strings.ToLower(orderID)
}

func adminOrderMessages(r *http.Request) (string, string) {
	switch r.URL.Query().Get("ok") {
	case "producao":
		return "Status de producao atualizado.", ""
	case "envio":
		return "Status de envio atualizado.", ""
	}
	if r.URL.Query().Get("erro") == "transicao" {
		return "", "Status atualizado por outra operacao ou transicao invalida. Recarregue e tente novamente."
	}

	return "", ""
}
