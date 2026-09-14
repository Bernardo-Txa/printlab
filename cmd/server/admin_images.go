package main

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	admindomain "github.com/Bernardo-Txa/printlab/internal/admin"
	"github.com/Bernardo-Txa/printlab/web/templates"
)

const adminMaxJSONBodyBytes = 64 << 10

type adminImageUploadRequest struct {
	VariantID   string `json:"variant_id"`
	ContentType string `json:"content_type"`
	FileSize    int64  `json:"file_size"`
	Filename    string `json:"filename"`
}

type adminImageFinalizeRequest struct {
	VariantID   string `json:"variant_id"`
	ObjectPath  string `json:"object_path"`
	ContentType string `json:"content_type"`
	FileSize    int64  `json:"file_size"`
	AltText     string `json:"alt_text"`
	SortOrder   int    `json:"sort_order"`
	IsPrimary   bool   `json:"is_primary"`
	Filename    string `json:"filename"`
}

func adminProductImagesHandler(service adminPanelService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setAdminPrivateHeaders(w)
		if _, ok := requireAdminSession(w, r, service); !ok {
			return
		}

		page, err := service.GetAdminProductImages(r.Context(), r.PathValue("productID"))
		if err != nil {
			renderAdminCatalogError(w, r, err)
			return
		}
		page.Message, page.ErrorMessage = adminImageMessages(r)

		renderHTML(w, r, http.StatusOK, templates.AdminProductImages(page))
	}
}

func adminImageUploadURLHandler(service adminPanelService, siteURL string) http.HandlerFunc {
	return adminImageAuthorizationHandler(service, siteURL, false)
}

func adminImageReplaceURLHandler(service adminPanelService, siteURL string) http.HandlerFunc {
	return adminImageAuthorizationHandler(service, siteURL, true)
}

func adminImageAuthorizationHandler(service adminPanelService, siteURL string, replacement bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setAdminPrivateHeaders(w)
		if !validAdminMutationSource(r, siteURL) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if _, ok := requireAdminSession(w, r, service); !ok {
			return
		}

		var req adminImageUploadRequest
		if !decodeAdminJSON(w, r, &req) {
			return
		}
		metadata := admindomain.AdminImageUploadMetadata{
			VariantID:   req.VariantID,
			ContentType: req.ContentType,
			FileSize:    req.FileSize,
			Filename:    req.Filename,
		}

		var authorization admindomain.AdminImageUploadAuthorization
		var err error
		if replacement {
			authorization, err = service.AuthorizeAdminImageReplacement(r.Context(), r.PathValue("productID"), r.PathValue("imageID"), metadata)
		} else {
			authorization, err = service.AuthorizeAdminImageUpload(r.Context(), r.PathValue("productID"), metadata)
		}
		if err != nil {
			handleAdminImageJSONError(w, err)
			return
		}

		writeAdminJSON(w, http.StatusOK, map[string]any{
			"upload_url":         authorization.UploadURL,
			"object_path":        authorization.ObjectPath,
			"content_type":       authorization.ContentType,
			"max_file_size":      authorization.MaxFileSizeBytes,
			"max_file_size_text": admindomain.FormatAdminFileSize(authorization.MaxFileSizeBytes),
		})
	}
}

func adminImageFinalizeHandler(service adminPanelService, siteURL string) http.HandlerFunc {
	return adminImageFinalizeJSONHandler(service, siteURL, false)
}

func adminImageReplaceFinalizeHandler(service adminPanelService, siteURL string) http.HandlerFunc {
	return adminImageFinalizeJSONHandler(service, siteURL, true)
}

func adminImageFinalizeJSONHandler(service adminPanelService, siteURL string, replacement bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setAdminPrivateHeaders(w)
		if !validAdminMutationSource(r, siteURL) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if _, ok := requireAdminSession(w, r, service); !ok {
			return
		}

		var req adminImageFinalizeRequest
		if !decodeAdminJSON(w, r, &req) {
			return
		}
		input := admindomain.AdminImageFinalizeInput{
			VariantID:   req.VariantID,
			ObjectPath:  req.ObjectPath,
			ContentType: req.ContentType,
			FileSize:    req.FileSize,
			AltText:     req.AltText,
			SortOrder:   req.SortOrder,
			IsPrimary:   req.IsPrimary,
			Filename:    req.Filename,
		}

		productID := r.PathValue("productID")
		var err error
		if replacement {
			err = service.FinalizeAdminImageReplacement(r.Context(), productID, r.PathValue("imageID"), input)
		} else {
			_, err = service.FinalizeAdminImageUpload(r.Context(), productID, input)
		}
		if err != nil {
			handleAdminImageJSONError(w, err)
			return
		}

		writeAdminJSON(w, http.StatusOK, map[string]any{
			"redirect_url": adminProductImagesPath(productID) + "?ok=imagem",
		})
	}
}

func adminImageRemoveHandler(service adminPanelService, siteURL string) http.HandlerFunc {
	return adminImageFormMutationHandler(service, siteURL, func(r *http.Request) error {
		return service.RemoveAdminProductImage(r.Context(), r.PathValue("productID"), r.PathValue("imageID"))
	}, "removida")
}

func adminImageOrderHandler(service adminPanelService, siteURL string) http.HandlerFunc {
	return adminImageFormMutationHandler(service, siteURL, func(r *http.Request) error {
		return service.UpdateAdminProductImageOrder(r.Context(), r.PathValue("productID"), r.PathValue("imageID"), r.PostFormValue("sort_order"))
	}, "ordem")
}

func adminImagePrimaryHandler(service adminPanelService, siteURL string) http.HandlerFunc {
	return adminImageFormMutationHandler(service, siteURL, func(r *http.Request) error {
		return service.MarkAdminProductImagePrimary(r.Context(), r.PathValue("productID"), r.PathValue("imageID"))
	}, "principal")
}

func adminImageFormMutationHandler(service adminPanelService, siteURL string, mutate func(*http.Request) error, okCode string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setAdminPrivateHeaders(w)
		if !validAdminMutationSource(r, siteURL) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if _, ok := requireAdminSession(w, r, service); !ok {
			return
		}
		if err := parseAdminForm(w, r); err != nil {
			writeInvalidBodyError(w, "invalid admin request", err)
			return
		}

		productID := r.PathValue("productID")
		if err := mutate(r); err != nil {
			if errors.Is(err, admindomain.ErrInvalidCatalogID) || errors.Is(err, admindomain.ErrCatalogNotFound) {
				http.NotFound(w, r)
				return
			}
			if errors.Is(err, admindomain.ErrValidation) {
				http.Redirect(w, r, adminProductImagesPath(productID)+"?erro=validacao", http.StatusSeeOther)
				return
			}
			log.Print("admin image mutation failed")
			http.Redirect(w, r, adminProductImagesPath(productID)+"?erro=storage", http.StatusSeeOther)
			return
		}

		http.Redirect(w, r, adminProductImagesPath(productID)+"?ok="+okCode, http.StatusSeeOther)
	}
}

func decodeAdminJSON(w http.ResponseWriter, r *http.Request, dest any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, adminMaxJSONBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dest); err != nil {
		if isMaxBytesError(err) {
			writeAdminJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "invalid_request"})
			return false
		}
		writeAdminJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request"})
		return false
	}

	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		if isMaxBytesError(err) {
			writeAdminJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "invalid_request"})
			return false
		}
		writeAdminJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request"})
		return false
	}

	return true
}

func handleAdminImageJSONError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, admindomain.ErrInvalidCatalogID), errors.Is(err, admindomain.ErrCatalogNotFound):
		writeAdminJSON(w, http.StatusNotFound, map[string]string{"error": "not_found"})
	case errors.Is(err, admindomain.ErrInvalidImageType):
		writeAdminJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "invalid_image_type"})
	case errors.Is(err, admindomain.ErrImageTooLarge):
		writeAdminJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "image_too_large"})
	case errors.Is(err, admindomain.ErrInvalidImagePath), errors.Is(err, admindomain.ErrValidation):
		writeAdminJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "invalid_image"})
	default:
		log.Print("admin image storage unavailable")
		writeAdminJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "storage_unavailable"})
	}
}

func writeAdminJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func adminProductImagesPath(productID string) string {
	return "/admin/produtos/" + strings.ToLower(productID) + "/imagens"
}

func adminImageMessages(r *http.Request) (string, string) {
	switch r.URL.Query().Get("ok") {
	case "imagem":
		return "Imagem salva.", ""
	case "removida":
		return "Imagem removida.", ""
	case "ordem":
		return "Ordem da imagem atualizada.", ""
	case "principal":
		return "Imagem principal atualizada.", ""
	}
	switch r.URL.Query().Get("erro") {
	case "validacao":
		return "", "Revise os dados da imagem e tente novamente."
	case "storage":
		return "", "Nao foi possivel concluir a operacao de Storage agora."
	}

	return "", ""
}
