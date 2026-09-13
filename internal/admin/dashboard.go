package admin

const (
	OrderStatusPendingPayment    = "pending_payment"
	OrderStatusPaid              = "paid"
	ProductionStatusWaiting      = "waiting"
	ProductionStatusInProduction = "in_production"
	ProductionStatusCompleted    = "completed"
	ShippingStatusWaiting        = "waiting"
	ShippingStatusPreparing      = "preparing"
	ShippingStatusShipped        = "shipped"
	ShippingStatusDelivered      = "delivered"
)

func DashboardFromOrderStatuses(records []OrderStatusSnapshot) Dashboard {
	var dashboard Dashboard
	for _, record := range records {
		if record.OrderStatus == OrderStatusPendingPayment {
			dashboard.PendingPayment++
			continue
		}
		if record.OrderStatus != OrderStatusPaid {
			continue
		}
		switch record.ProductionStatus {
		case ProductionStatusWaiting:
			dashboard.PaidWaitingProduction++
		case ProductionStatusInProduction:
			dashboard.InProduction++
		case ProductionStatusCompleted:
			if record.ShippingStatus == ShippingStatusWaiting {
				dashboard.WaitingShipment++
			}
		}
	}

	return dashboard
}
