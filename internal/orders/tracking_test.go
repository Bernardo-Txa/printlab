package orders

import (
	"testing"
	"time"
)

func TestTrackingPageStatusLabels(t *testing.T) {
	createdAt := time.Date(2026, 9, 12, 14, 30, 0, 0, time.UTC)
	tests := []struct {
		name       string
		payment    string
		production string
		shipping   string
		wantPay    string
		wantProd   string
		wantShip   string
	}{
		{
			name:       "pending payment waiting",
			payment:    StatusPendingPayment,
			production: ProductionStatusWaiting,
			shipping:   ShippingStatusWaiting,
			wantPay:    "Aguardando pagamento",
			wantProd:   "Sera iniciada apos a confirmacao do pagamento",
			wantShip:   "Sera preparado apos a producao",
		},
		{
			name:       "paid waiting",
			payment:    StatusPaid,
			production: ProductionStatusWaiting,
			shipping:   ShippingStatusWaiting,
			wantPay:    "Pagamento confirmado",
			wantProd:   "Aguardando producao",
			wantShip:   "Aguardando producao",
		},
		{
			name:       "paid in production",
			payment:    StatusPaid,
			production: ProductionStatusInProduction,
			shipping:   ShippingStatusWaiting,
			wantPay:    "Pagamento confirmado",
			wantProd:   "Em producao",
			wantShip:   "Aguardando conclusao da producao",
		},
		{
			name:       "paid completed waiting shipping",
			payment:    StatusPaid,
			production: ProductionStatusCompleted,
			shipping:   ShippingStatusWaiting,
			wantPay:    "Pagamento confirmado",
			wantProd:   "Producao concluida",
			wantShip:   "Aguardando preparacao do envio",
		},
		{
			name:       "shipping preparing",
			payment:    StatusPaid,
			production: ProductionStatusCompleted,
			shipping:   ShippingStatusPreparing,
			wantPay:    "Pagamento confirmado",
			wantProd:   "Producao concluida",
			wantShip:   "Preparando envio",
		},
		{
			name:       "shipping shipped",
			payment:    StatusPaid,
			production: ProductionStatusCompleted,
			shipping:   ShippingStatusShipped,
			wantPay:    "Pagamento confirmado",
			wantProd:   "Producao concluida",
			wantShip:   "Enviado",
		},
		{
			name:       "shipping delivered",
			payment:    StatusPaid,
			production: ProductionStatusCompleted,
			shipping:   ShippingStatusDelivered,
			wantPay:    "Pagamento confirmado",
			wantProd:   "Producao concluida",
			wantShip:   "Entregue",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page := TrackingPageFromRecord(TrackingRecord{
				OrderNumber:      1004,
				Status:           tt.payment,
				CreatedAt:        createdAt,
				ProductionStatus: tt.production,
				ShippingStatus:   tt.shipping,
				ServiceName:      "PAC",
				CarrierName:      "Correios",
			})

			if page.PaymentStatusLabel != tt.wantPay {
				t.Fatalf("expected payment label %q, got %q", tt.wantPay, page.PaymentStatusLabel)
			}
			if page.ProductionStatusLabel != tt.wantProd {
				t.Fatalf("expected production label %q, got %q", tt.wantProd, page.ProductionStatusLabel)
			}
			if page.ShippingStatusLabel != tt.wantShip {
				t.Fatalf("expected shipping label %q, got %q", tt.wantShip, page.ShippingStatusLabel)
			}
			if page.ShippingServiceLabel != "Correios - PAC" {
				t.Fatalf("expected commercial shipping label, got %q", page.ShippingServiceLabel)
			}
			if len(page.Steps) != 3 {
				t.Fatalf("expected three tracking steps, got %d", len(page.Steps))
			}
		})
	}
}

func TestTrackingIDValidationAndPath(t *testing.T) {
	trackingID := "AAAAAAAA-BBBB-CCCC-DDDD-EEEEEEEEEEEE"
	if !ValidTrackingID(trackingID) {
		t.Fatal("expected valid UUID tracking id")
	}
	if got := TrackingPath(trackingID); got != "/acompanhar/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee" {
		t.Fatalf("expected canonical tracking path, got %q", got)
	}
	for _, invalid := range []string{"1004", "not-a-uuid", "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeeeg"} {
		if ValidTrackingID(invalid) {
			t.Fatalf("expected %q to be invalid", invalid)
		}
		if TrackingPath(invalid) != "" {
			t.Fatalf("expected empty tracking path for %q", invalid)
		}
	}
}
