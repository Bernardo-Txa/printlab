package admin

import (
	"errors"
	"testing"
	"time"
)

func TestValidateProductionTransitionRequiresPaidOrderAndAdjacentProgress(t *testing.T) {
	tests := []struct {
		name     string
		snapshot OrderStatusSnapshot
		target   string
		wantErr  error
	}{
		{
			name: "pending payment cannot start production",
			snapshot: OrderStatusSnapshot{
				OrderStatus:      OrderStatusPendingPayment,
				ProductionStatus: ProductionStatusWaiting,
				ShippingStatus:   ShippingStatusWaiting,
			},
			target:  ProductionStatusInProduction,
			wantErr: ErrInvalidTransition,
		},
		{
			name: "paid waiting can start production",
			snapshot: OrderStatusSnapshot{
				OrderStatus:      OrderStatusPaid,
				ProductionStatus: ProductionStatusWaiting,
				ShippingStatus:   ShippingStatusWaiting,
			},
			target: ProductionStatusInProduction,
		},
		{
			name: "paid waiting cannot complete directly",
			snapshot: OrderStatusSnapshot{
				OrderStatus:      OrderStatusPaid,
				ProductionStatus: ProductionStatusWaiting,
				ShippingStatus:   ShippingStatusWaiting,
			},
			target:  ProductionStatusCompleted,
			wantErr: ErrInvalidTransition,
		},
		{
			name: "paid in production can complete",
			snapshot: OrderStatusSnapshot{
				OrderStatus:      OrderStatusPaid,
				ProductionStatus: ProductionStatusInProduction,
				ShippingStatus:   ShippingStatusWaiting,
			},
			target: ProductionStatusCompleted,
		},
		{
			name: "completed cannot restart",
			snapshot: OrderStatusSnapshot{
				OrderStatus:      OrderStatusPaid,
				ProductionStatus: ProductionStatusCompleted,
				ShippingStatus:   ShippingStatusWaiting,
			},
			target:  ProductionStatusInProduction,
			wantErr: ErrInvalidTransition,
		},
		{
			name: "same status is a conflict",
			snapshot: OrderStatusSnapshot{
				OrderStatus:      OrderStatusPaid,
				ProductionStatus: ProductionStatusInProduction,
				ShippingStatus:   ShippingStatusWaiting,
			},
			target:  ProductionStatusInProduction,
			wantErr: ErrTransitionConflict,
		},
		{
			name: "shipping already moving blocks production mutation",
			snapshot: OrderStatusSnapshot{
				OrderStatus:      OrderStatusPaid,
				ProductionStatus: ProductionStatusCompleted,
				ShippingStatus:   ShippingStatusPreparing,
			},
			target:  ProductionStatusInProduction,
			wantErr: ErrInvalidTransition,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProductionTransition(tt.snapshot, tt.target)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestValidateShippingTransitionRequiresCompletedProductionAndAdjacentProgress(t *testing.T) {
	tests := []struct {
		name     string
		snapshot OrderStatusSnapshot
		target   string
		wantErr  error
	}{
		{
			name: "pending payment cannot prepare shipment",
			snapshot: OrderStatusSnapshot{
				OrderStatus:      OrderStatusPendingPayment,
				ProductionStatus: ProductionStatusWaiting,
				ShippingStatus:   ShippingStatusWaiting,
			},
			target:  ShippingStatusPreparing,
			wantErr: ErrInvalidTransition,
		},
		{
			name: "in production cannot prepare shipment",
			snapshot: OrderStatusSnapshot{
				OrderStatus:      OrderStatusPaid,
				ProductionStatus: ProductionStatusInProduction,
				ShippingStatus:   ShippingStatusWaiting,
			},
			target:  ShippingStatusPreparing,
			wantErr: ErrInvalidTransition,
		},
		{
			name: "completed production can prepare shipment",
			snapshot: OrderStatusSnapshot{
				OrderStatus:      OrderStatusPaid,
				ProductionStatus: ProductionStatusCompleted,
				ShippingStatus:   ShippingStatusWaiting,
			},
			target: ShippingStatusPreparing,
		},
		{
			name: "preparing can be shipped",
			snapshot: OrderStatusSnapshot{
				OrderStatus:      OrderStatusPaid,
				ProductionStatus: ProductionStatusCompleted,
				ShippingStatus:   ShippingStatusPreparing,
			},
			target: ShippingStatusShipped,
		},
		{
			name: "shipped can be delivered",
			snapshot: OrderStatusSnapshot{
				OrderStatus:      OrderStatusPaid,
				ProductionStatus: ProductionStatusCompleted,
				ShippingStatus:   ShippingStatusShipped,
			},
			target: ShippingStatusDelivered,
		},
		{
			name: "waiting cannot be shipped directly",
			snapshot: OrderStatusSnapshot{
				OrderStatus:      OrderStatusPaid,
				ProductionStatus: ProductionStatusCompleted,
				ShippingStatus:   ShippingStatusWaiting,
			},
			target:  ShippingStatusShipped,
			wantErr: ErrInvalidTransition,
		},
		{
			name: "preparing cannot be delivered directly",
			snapshot: OrderStatusSnapshot{
				OrderStatus:      OrderStatusPaid,
				ProductionStatus: ProductionStatusCompleted,
				ShippingStatus:   ShippingStatusPreparing,
			},
			target:  ShippingStatusDelivered,
			wantErr: ErrInvalidTransition,
		},
		{
			name: "delivered is terminal",
			snapshot: OrderStatusSnapshot{
				OrderStatus:      OrderStatusPaid,
				ProductionStatus: ProductionStatusCompleted,
				ShippingStatus:   ShippingStatusDelivered,
			},
			target:  ShippingStatusShipped,
			wantErr: ErrInvalidTransition,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateShippingTransition(tt.snapshot, tt.target)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestPrepareOrderListPageAppliesLabelsAndPagination(t *testing.T) {
	items := make([]OrderListItem, 0, OrderPageSize+1)
	for i := 0; i < OrderPageSize+1; i++ {
		items = append(items, OrderListItem{
			ID:               "11111111-1111-1111-1111-111111111111",
			OrderNumber:      1001 + int64(i),
			OrderStatus:      OrderStatusPaid,
			ProductionStatus: ProductionStatusCompleted,
			ShippingStatus:   ShippingStatusWaiting,
			CreatedAt:        time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC),
			TotalCents:       12345,
		})
	}

	page := PrepareOrderListPage(OrderListFilter{Status: OrderListStatusWaitingShipment, Page: 2}, items)

	if len(page.Orders) != OrderPageSize || !page.HasNext || !page.HasPrevious {
		t.Fatalf("unexpected pagination: %#v", page)
	}
	if page.StatusLabel != "Aguardando envio" {
		t.Fatalf("expected status label, got %q", page.StatusLabel)
	}
	if page.Orders[0].OrderNumberLabel != "#1001" || page.Orders[0].TotalBRL == "" || page.Orders[0].DetailURL == "" {
		t.Fatalf("expected prepared order item, got %#v", page.Orders[0])
	}
}
