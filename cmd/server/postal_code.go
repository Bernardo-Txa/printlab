package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/Bernardo-Txa/printlab/internal/customers"
)

type postalCodeLookupService interface {
	Lookup(ctx context.Context, postalCode string) (customers.PostalCodeAddress, error)
}

type postalCodeLookupResponse struct {
	Street   string `json:"street"`
	District string `json:"district"`
	City     string `json:"city"`
	State    string `json:"state"`
}

func postalCodeLookupHandler(service postalCodeLookupService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setCheckoutPrivateCache(w)

		if service == nil {
			log.Print("cep lookup failed reason=not_configured")
			writePostalCodeLookupResponse(w, http.StatusServiceUnavailable, customers.PostalCodeAddress{})
			return
		}

		address, err := service.Lookup(r.Context(), r.PathValue("cep"))
		if err != nil {
			handlePostalCodeLookupError(w, err)
			return
		}

		writePostalCodeLookupResponse(w, http.StatusOK, address)
	}
}

func handlePostalCodeLookupError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, customers.ErrInvalidDetails):
		writePostalCodeLookupResponse(w, http.StatusBadRequest, customers.PostalCodeAddress{})
	case errors.Is(err, customers.ErrPostalCodeNotFound):
		writePostalCodeLookupResponse(w, http.StatusNotFound, customers.PostalCodeAddress{})
	default:
		logPostalCodeLookupFailure(err)
		writePostalCodeLookupResponse(w, http.StatusServiceUnavailable, customers.PostalCodeAddress{})
	}
}

func writePostalCodeLookupResponse(w http.ResponseWriter, statusCode int, address customers.PostalCodeAddress) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(postalCodeLookupResponse{
		Street:   address.Street,
		District: address.District,
		City:     address.City,
		State:    address.State,
	})
}

func logPostalCodeLookupFailure(err error) {
	var lookupErr *customers.PostalCodeLookupError
	if errors.As(err, &lookupErr) {
		if lookupErr.StatusCode > 0 {
			log.Printf("cep lookup failed reason=%s status=%d", lookupErr.Reason, lookupErr.StatusCode)
			return
		}

		log.Printf("cep lookup failed reason=%s", lookupErr.Reason)
		return
	}

	log.Print("cep lookup failed reason=unavailable")
}
