package main

import (
	"errors"
	"log"
	"net/http"

	admindomain "github.com/Bernardo-Txa/printlab/internal/admin"
	"github.com/Bernardo-Txa/printlab/web/templates"
	"github.com/a-h/templ"
)

func adminProductsHandler(service adminPanelService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setAdminPrivateHeaders(w)
		if _, ok := requireAdminSession(w, r, service); !ok {
			return
		}
		page, err := service.ListAdminProducts(r.Context(), admindomain.AdminProductListFilter{
			Status: r.URL.Query().Get("status"),
			Query:  r.URL.Query().Get("q"),
		})
		if err != nil {
			renderAdminCatalogError(w, r, err)
			return
		}

		renderHTML(w, r, http.StatusOK, templates.AdminProducts(page))
	}
}

func adminNewProductHandler(service adminPanelService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setAdminPrivateHeaders(w)
		if _, ok := requireAdminSession(w, r, service); !ok {
			return
		}
		page, err := service.NewAdminProduct(r.Context())
		if err != nil {
			renderAdminCatalogError(w, r, err)
			return
		}

		renderHTML(w, r, http.StatusOK, templates.AdminProductForm(page))
	}
}

func adminCreateProductHandler(service adminPanelService, siteURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setAdminPrivateHeaders(w)
		if !validMutationSource(r, siteURL) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if _, ok := requireAdminSession(w, r, service); !ok {
			return
		}
		if err := parseAdminForm(w, r); err != nil {
			http.Error(w, "invalid admin request", http.StatusBadRequest)
			return
		}
		id, page, err := service.CreateAdminProduct(r.Context(), adminProductFormFromRequest(r))
		if err != nil {
			if errors.Is(err, admindomain.ErrValidation) || errors.Is(err, admindomain.ErrDuplicateSlug) {
				renderHTML(w, r, http.StatusUnprocessableEntity, templates.AdminProductForm(page))
				return
			}
			renderAdminCatalogError(w, r, err)
			return
		}

		http.Redirect(w, r, "/admin/produtos/"+id+"?ok=salvo", http.StatusSeeOther)
	}
}

func adminProductDetailHandler(service adminPanelService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setAdminPrivateHeaders(w)
		if _, ok := requireAdminSession(w, r, service); !ok {
			return
		}
		page, err := service.GetAdminProduct(r.Context(), r.PathValue("productID"))
		if err != nil {
			renderAdminCatalogError(w, r, err)
			return
		}
		page.Message = adminCatalogMessage(r)

		renderHTML(w, r, http.StatusOK, templates.AdminProductForm(page))
	}
}

func adminUpdateProductHandler(service adminPanelService, siteURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setAdminPrivateHeaders(w)
		if !validMutationSource(r, siteURL) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if _, ok := requireAdminSession(w, r, service); !ok {
			return
		}
		if err := parseAdminForm(w, r); err != nil {
			http.Error(w, "invalid admin request", http.StatusBadRequest)
			return
		}
		productID := r.PathValue("productID")
		page, err := service.UpdateAdminProduct(r.Context(), productID, adminProductFormFromRequest(r))
		if err != nil {
			if errors.Is(err, admindomain.ErrValidation) || errors.Is(err, admindomain.ErrDuplicateSlug) {
				renderHTML(w, r, http.StatusUnprocessableEntity, templates.AdminProductForm(page))
				return
			}
			renderAdminCatalogError(w, r, err)
			return
		}

		http.Redirect(w, r, "/admin/produtos/"+productID+"?ok=salvo", http.StatusSeeOther)
	}
}

func adminCategoriesHandler(service adminPanelService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setAdminPrivateHeaders(w)
		if _, ok := requireAdminSession(w, r, service); !ok {
			return
		}
		page, err := service.ListAdminCategories(r.Context())
		if err != nil {
			renderAdminCatalogError(w, r, err)
			return
		}

		renderHTML(w, r, http.StatusOK, templates.AdminCategories(page))
	}
}

func adminNewCategoryHandler(service adminPanelService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setAdminPrivateHeaders(w)
		if _, ok := requireAdminSession(w, r, service); !ok {
			return
		}

		renderHTML(w, r, http.StatusOK, templates.AdminCategoryForm(service.NewAdminCategory()))
	}
}

func adminCreateCategoryHandler(service adminPanelService, siteURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setAdminPrivateHeaders(w)
		if !validMutationSource(r, siteURL) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if _, ok := requireAdminSession(w, r, service); !ok {
			return
		}
		if err := parseAdminForm(w, r); err != nil {
			http.Error(w, "invalid admin request", http.StatusBadRequest)
			return
		}
		id, page, err := service.CreateAdminCategory(r.Context(), adminCategoryFormFromRequest(r))
		handleSimpleAdminCreate(w, r, err, "/admin/categorias/"+id+"?ok=salvo", templates.AdminCategoryForm(page))
	}
}

func adminCategoryDetailHandler(service adminPanelService) http.HandlerFunc {
	return adminSimpleGetHandler(service, func(r *http.Request) (templComponent, error) {
		page, err := service.GetAdminCategory(r.Context(), r.PathValue("categoryID"))
		page.Message = adminCatalogMessage(r)
		return templates.AdminCategoryForm(page), err
	})
}

func adminUpdateCategoryHandler(service adminPanelService, siteURL string) http.HandlerFunc {
	return adminSimplePostHandler(service, siteURL, func(r *http.Request) (templComponent, string, error) {
		id := r.PathValue("categoryID")
		page, err := service.UpdateAdminCategory(r.Context(), id, adminCategoryFormFromRequest(r))
		return templates.AdminCategoryForm(page), "/admin/categorias/" + id + "?ok=salvo", err
	})
}

func adminNewVariantHandler(service adminPanelService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setAdminPrivateHeaders(w)
		if _, ok := requireAdminSession(w, r, service); !ok {
			return
		}
		page, err := service.NewAdminVariant(r.Context(), r.PathValue("productID"))
		if err != nil {
			renderAdminCatalogError(w, r, err)
			return
		}

		renderHTML(w, r, http.StatusOK, templates.AdminVariantForm(page))
	}
}

func adminCreateVariantHandler(service adminPanelService, siteURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setAdminPrivateHeaders(w)
		if !validMutationSource(r, siteURL) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if _, ok := requireAdminSession(w, r, service); !ok {
			return
		}
		if err := parseAdminForm(w, r); err != nil {
			http.Error(w, "invalid admin request", http.StatusBadRequest)
			return
		}
		productID := r.PathValue("productID")
		id, page, err := service.CreateAdminVariant(r.Context(), productID, adminVariantFormFromRequest(r))
		if err != nil {
			if adminCatalogFormError(err) {
				renderHTML(w, r, http.StatusUnprocessableEntity, templates.AdminVariantForm(page))
				return
			}
			renderAdminCatalogError(w, r, err)
			return
		}

		http.Redirect(w, r, "/admin/produtos/"+productID+"/variantes/"+id+"?ok=salvo", http.StatusSeeOther)
	}
}

func adminVariantDetailHandler(service adminPanelService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setAdminPrivateHeaders(w)
		if _, ok := requireAdminSession(w, r, service); !ok {
			return
		}
		page, err := service.GetAdminVariant(r.Context(), r.PathValue("productID"), r.PathValue("variantID"))
		if err != nil {
			renderAdminCatalogError(w, r, err)
			return
		}
		page.Message = adminCatalogMessage(r)

		renderHTML(w, r, http.StatusOK, templates.AdminVariantForm(page))
	}
}

func adminUpdateVariantHandler(service adminPanelService, siteURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setAdminPrivateHeaders(w)
		if !validMutationSource(r, siteURL) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if _, ok := requireAdminSession(w, r, service); !ok {
			return
		}
		if err := parseAdminForm(w, r); err != nil {
			http.Error(w, "invalid admin request", http.StatusBadRequest)
			return
		}
		productID := r.PathValue("productID")
		variantID := r.PathValue("variantID")
		page, err := service.UpdateAdminVariant(r.Context(), productID, variantID, adminVariantFormFromRequest(r))
		if err != nil {
			if adminCatalogFormError(err) {
				renderHTML(w, r, http.StatusUnprocessableEntity, templates.AdminVariantForm(page))
				return
			}
			renderAdminCatalogError(w, r, err)
			return
		}

		http.Redirect(w, r, "/admin/produtos/"+productID+"/variantes/"+variantID+"?ok=salvo", http.StatusSeeOther)
	}
}

func adminAddRecipeComponentHandler(service adminPanelService, siteURL string) http.HandlerFunc {
	return adminRecipeMutationHandler(service, siteURL, func(r *http.Request) error {
		return service.AddAdminRecipeComponent(r.Context(), r.PathValue("productID"), r.PathValue("variantID"), adminRecipeFormFromRequest(r))
	})
}

func adminUpdateRecipeComponentHandler(service adminPanelService, siteURL string) http.HandlerFunc {
	return adminRecipeMutationHandler(service, siteURL, func(r *http.Request) error {
		return service.UpdateAdminRecipeComponent(r.Context(), r.PathValue("productID"), r.PathValue("variantID"), r.PathValue("componentID"), adminRecipeFormFromRequest(r))
	})
}

func adminRemoveRecipeComponentHandler(service adminPanelService, siteURL string) http.HandlerFunc {
	return adminRecipeMutationHandler(service, siteURL, func(r *http.Request) error {
		return service.RemoveAdminRecipeComponent(r.Context(), r.PathValue("productID"), r.PathValue("variantID"), r.PathValue("componentID"))
	})
}

func adminMaterialsHandler(service adminPanelService) http.HandlerFunc {
	return adminSimpleListHandler(service, func(r *http.Request) (templComponent, error) {
		page, err := service.ListAdminMaterials(r.Context())
		return templates.AdminMaterials(page), err
	})
}

func adminNewMaterialHandler(service adminPanelService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setAdminPrivateHeaders(w)
		if _, ok := requireAdminSession(w, r, service); !ok {
			return
		}

		renderHTML(w, r, http.StatusOK, templates.AdminMaterialForm(service.NewAdminMaterial()))
	}
}

func adminCreateMaterialHandler(service adminPanelService, siteURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setAdminPrivateHeaders(w)
		if !validMutationSource(r, siteURL) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if _, ok := requireAdminSession(w, r, service); !ok {
			return
		}
		if err := parseAdminForm(w, r); err != nil {
			http.Error(w, "invalid admin request", http.StatusBadRequest)
			return
		}
		id, page, err := service.CreateAdminMaterial(r.Context(), adminMaterialFormFromRequest(r))
		handleSimpleAdminCreate(w, r, err, "/admin/materiais/"+id+"?ok=salvo", templates.AdminMaterialForm(page))
	}
}

func adminMaterialDetailHandler(service adminPanelService) http.HandlerFunc {
	return adminSimpleGetHandler(service, func(r *http.Request) (templComponent, error) {
		page, err := service.GetAdminMaterial(r.Context(), r.PathValue("materialID"))
		page.Message = adminCatalogMessage(r)
		return templates.AdminMaterialForm(page), err
	})
}

func adminUpdateMaterialHandler(service adminPanelService, siteURL string) http.HandlerFunc {
	return adminSimplePostHandler(service, siteURL, func(r *http.Request) (templComponent, string, error) {
		id := r.PathValue("materialID")
		page, err := service.UpdateAdminMaterial(r.Context(), id, adminMaterialFormFromRequest(r))
		return templates.AdminMaterialForm(page), "/admin/materiais/" + id + "?ok=salvo", err
	})
}

func adminColorsHandler(service adminPanelService) http.HandlerFunc {
	return adminSimpleListHandler(service, func(r *http.Request) (templComponent, error) {
		page, err := service.ListAdminColors(r.Context())
		return templates.AdminColors(page), err
	})
}

func adminNewColorHandler(service adminPanelService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setAdminPrivateHeaders(w)
		if _, ok := requireAdminSession(w, r, service); !ok {
			return
		}

		renderHTML(w, r, http.StatusOK, templates.AdminColorForm(service.NewAdminColor()))
	}
}

func adminCreateColorHandler(service adminPanelService, siteURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setAdminPrivateHeaders(w)
		if !validMutationSource(r, siteURL) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if _, ok := requireAdminSession(w, r, service); !ok {
			return
		}
		if err := parseAdminForm(w, r); err != nil {
			http.Error(w, "invalid admin request", http.StatusBadRequest)
			return
		}
		id, page, err := service.CreateAdminColor(r.Context(), adminColorFormFromRequest(r))
		handleSimpleAdminCreate(w, r, err, "/admin/cores/"+id+"?ok=salvo", templates.AdminColorForm(page))
	}
}

func adminColorDetailHandler(service adminPanelService) http.HandlerFunc {
	return adminSimpleGetHandler(service, func(r *http.Request) (templComponent, error) {
		page, err := service.GetAdminColor(r.Context(), r.PathValue("colorID"))
		page.Message = adminCatalogMessage(r)
		return templates.AdminColorForm(page), err
	})
}

func adminUpdateColorHandler(service adminPanelService, siteURL string) http.HandlerFunc {
	return adminSimplePostHandler(service, siteURL, func(r *http.Request) (templComponent, string, error) {
		id := r.PathValue("colorID")
		page, err := service.UpdateAdminColor(r.Context(), id, adminColorFormFromRequest(r))
		return templates.AdminColorForm(page), "/admin/cores/" + id + "?ok=salvo", err
	})
}

func adminBoxesHandler(service adminPanelService) http.HandlerFunc {
	return adminSimpleListHandler(service, func(r *http.Request) (templComponent, error) {
		page, err := service.ListAdminBoxes(r.Context())
		return templates.AdminBoxes(page), err
	})
}

func adminNewBoxHandler(service adminPanelService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setAdminPrivateHeaders(w)
		if _, ok := requireAdminSession(w, r, service); !ok {
			return
		}

		renderHTML(w, r, http.StatusOK, templates.AdminBoxForm(service.NewAdminBox()))
	}
}

func adminCreateBoxHandler(service adminPanelService, siteURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setAdminPrivateHeaders(w)
		if !validMutationSource(r, siteURL) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if _, ok := requireAdminSession(w, r, service); !ok {
			return
		}
		if err := parseAdminForm(w, r); err != nil {
			http.Error(w, "invalid admin request", http.StatusBadRequest)
			return
		}
		id, page, err := service.CreateAdminBox(r.Context(), adminBoxFormFromRequest(r))
		handleSimpleAdminCreate(w, r, err, "/admin/caixas/"+id+"?ok=salvo", templates.AdminBoxForm(page))
	}
}

func adminBoxDetailHandler(service adminPanelService) http.HandlerFunc {
	return adminSimpleGetHandler(service, func(r *http.Request) (templComponent, error) {
		page, err := service.GetAdminBox(r.Context(), r.PathValue("boxID"))
		page.Message = adminCatalogMessage(r)
		return templates.AdminBoxForm(page), err
	})
}

func adminUpdateBoxHandler(service adminPanelService, siteURL string) http.HandlerFunc {
	return adminSimplePostHandler(service, siteURL, func(r *http.Request) (templComponent, string, error) {
		id := r.PathValue("boxID")
		page, err := service.UpdateAdminBox(r.Context(), id, adminBoxFormFromRequest(r))
		return templates.AdminBoxForm(page), "/admin/caixas/" + id + "?ok=salvo", err
	})
}

type templComponent = templ.Component

func adminSimpleListHandler(service adminPanelService, load func(r *http.Request) (templComponent, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setAdminPrivateHeaders(w)
		if _, ok := requireAdminSession(w, r, service); !ok {
			return
		}
		component, err := load(r)
		if err != nil {
			renderAdminCatalogError(w, r, err)
			return
		}

		renderHTML(w, r, http.StatusOK, component)
	}
}

func adminSimpleGetHandler(service adminPanelService, load func(r *http.Request) (templComponent, error)) http.HandlerFunc {
	return adminSimpleListHandler(service, load)
}

func adminSimplePostHandler(service adminPanelService, siteURL string, save func(r *http.Request) (templComponent, string, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setAdminPrivateHeaders(w)
		if !validMutationSource(r, siteURL) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if _, ok := requireAdminSession(w, r, service); !ok {
			return
		}
		if err := parseAdminForm(w, r); err != nil {
			http.Error(w, "invalid admin request", http.StatusBadRequest)
			return
		}
		component, redirectURL, err := save(r)
		if err != nil {
			if adminCatalogFormError(err) {
				renderHTML(w, r, http.StatusUnprocessableEntity, component)
				return
			}
			renderAdminCatalogError(w, r, err)
			return
		}

		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
	}
}

func adminRecipeMutationHandler(service adminPanelService, siteURL string, mutate func(r *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setAdminPrivateHeaders(w)
		if !validMutationSource(r, siteURL) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if _, ok := requireAdminSession(w, r, service); !ok {
			return
		}
		if err := parseAdminForm(w, r); err != nil {
			http.Error(w, "invalid admin request", http.StatusBadRequest)
			return
		}
		productID := r.PathValue("productID")
		variantID := r.PathValue("variantID")
		if err := mutate(r); err != nil {
			if errors.Is(err, admindomain.ErrInvalidCatalogID) || errors.Is(err, admindomain.ErrCatalogNotFound) {
				http.NotFound(w, r)
				return
			}
			if errors.Is(err, admindomain.ErrValidation) {
				page, pageErr := service.GetAdminVariant(r.Context(), productID, variantID)
				if pageErr != nil {
					renderAdminCatalogError(w, r, pageErr)
					return
				}
				page.RecipeForm = adminRecipeFormFromRequest(r)
				page.Message = "Revise os campos da receita."
				renderHTML(w, r, http.StatusUnprocessableEntity, templates.AdminVariantForm(page))
				return
			}
			renderAdminCatalogError(w, r, err)
			return
		}

		http.Redirect(w, r, "/admin/produtos/"+productID+"/variantes/"+variantID+"?ok=receita", http.StatusSeeOther)
	}
}

func handleSimpleAdminCreate(w http.ResponseWriter, r *http.Request, err error, redirectURL string, component templComponent) {
	if err != nil {
		if adminCatalogFormError(err) {
			renderHTML(w, r, http.StatusUnprocessableEntity, component)
			return
		}
		renderAdminCatalogError(w, r, err)
		return
	}

	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func renderAdminCatalogError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, admindomain.ErrInvalidCatalogID) || errors.Is(err, admindomain.ErrCatalogNotFound) {
		http.NotFound(w, r)
		return
	}

	log.Print("admin repository error")
	renderHTML(w, r, http.StatusServiceUnavailable, templates.AdminUnavailable())
}

func adminCatalogFormError(err error) bool {
	return errors.Is(err, admindomain.ErrValidation) ||
		errors.Is(err, admindomain.ErrDuplicateSlug) ||
		errors.Is(err, admindomain.ErrDuplicateSKU)
}

func adminCatalogMessage(r *http.Request) string {
	switch r.URL.Query().Get("ok") {
	case "salvo":
		return "Alteracoes salvas."
	case "receita":
		return "Receita atualizada."
	}

	return ""
}

func adminProductFormFromRequest(r *http.Request) admindomain.AdminProductForm {
	return admindomain.AdminProductForm{
		Name:             r.PostFormValue("name"),
		Slug:             r.PostFormValue("slug"),
		CategoryID:       r.PostFormValue("category_id"),
		ShortDescription: r.PostFormValue("short_description"),
		Description:      r.PostFormValue("description"),
		PriceBRL:         r.PostFormValue("price"),
		IsActive:         r.PostFormValue("is_active") == "1",
		IsFeatured:       r.PostFormValue("is_featured") == "1",
		UseShipping:      r.PostFormValue("use_shipping") == "1",
		ShippingWeightG:  r.PostFormValue("shipping_weight_g"),
		ShippingHeightMM: r.PostFormValue("shipping_height_mm"),
		ShippingWidthMM:  r.PostFormValue("shipping_width_mm"),
		ShippingLengthMM: r.PostFormValue("shipping_length_mm"),
	}
}

func adminCategoryFormFromRequest(r *http.Request) admindomain.AdminCategoryForm {
	return admindomain.AdminCategoryForm{
		Name:        r.PostFormValue("name"),
		Slug:        r.PostFormValue("slug"),
		Description: r.PostFormValue("description"),
		IsActive:    r.PostFormValue("is_active") == "1",
	}
}

func adminVariantFormFromRequest(r *http.Request) admindomain.AdminVariantForm {
	return admindomain.AdminVariantForm{
		Name:             r.PostFormValue("name"),
		Slug:             r.PostFormValue("slug"),
		SKU:              r.PostFormValue("sku"),
		PriceBRL:         r.PostFormValue("price"),
		IsActive:         r.PostFormValue("is_active") == "1",
		IsDefault:        r.PostFormValue("is_default") == "1",
		SortOrder:        r.PostFormValue("sort_order"),
		PrintTimeMinutes: r.PostFormValue("print_time_minutes"),
		UseShipping:      r.PostFormValue("use_shipping") == "1",
		ShippingWeightG:  r.PostFormValue("shipping_weight_g"),
		ShippingHeightMM: r.PostFormValue("shipping_height_mm"),
		ShippingWidthMM:  r.PostFormValue("shipping_width_mm"),
		ShippingLengthMM: r.PostFormValue("shipping_length_mm"),
	}
}

func adminRecipeFormFromRequest(r *http.Request) admindomain.AdminRecipeForm {
	return admindomain.AdminRecipeForm{
		MaterialID:      r.PostFormValue("material_id"),
		ColorID:         r.PostFormValue("color_id"),
		EstimatedWeight: r.PostFormValue("estimated_weight"),
		Label:           r.PostFormValue("label"),
		SortOrder:       r.PostFormValue("sort_order"),
	}
}

func adminMaterialFormFromRequest(r *http.Request) admindomain.AdminMaterialForm {
	return admindomain.AdminMaterialForm{
		Name:        r.PostFormValue("name"),
		Slug:        r.PostFormValue("slug"),
		Description: r.PostFormValue("description"),
		IsActive:    r.PostFormValue("is_active") == "1",
	}
}

func adminColorFormFromRequest(r *http.Request) admindomain.AdminColorForm {
	return admindomain.AdminColorForm{
		Name:     r.PostFormValue("name"),
		Slug:     r.PostFormValue("slug"),
		HexColor: r.PostFormValue("hex_color"),
		IsActive: r.PostFormValue("is_active") == "1",
	}
}

func adminBoxFormFromRequest(r *http.Request) admindomain.AdminBoxForm {
	return admindomain.AdminBoxForm{
		Name:             r.PostFormValue("name"),
		Slug:             r.PostFormValue("slug"),
		InternalHeightMM: r.PostFormValue("internal_height_mm"),
		InternalWidthMM:  r.PostFormValue("internal_width_mm"),
		InternalLengthMM: r.PostFormValue("internal_length_mm"),
		ExternalHeightMM: r.PostFormValue("external_height_mm"),
		ExternalWidthMM:  r.PostFormValue("external_width_mm"),
		ExternalLengthMM: r.PostFormValue("external_length_mm"),
		PackagingWeightG: r.PostFormValue("packaging_weight_g"),
		IsActive:         r.PostFormValue("is_active") == "1",
		SortOrder:        r.PostFormValue("sort_order"),
	}
}
