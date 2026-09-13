package admin

import (
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Bernardo-Txa/printlab/internal/products"
)

const (
	OrderListStatusAll               = "all"
	OrderListStatusPendingPayment    = "pending_payment"
	OrderListStatusWaitingProduction = "waiting_production"
	OrderListStatusInProduction      = "in_production"
	OrderListStatusWaitingShipment   = "waiting_shipment"
	OrderListStatusPreparingShipment = "preparing_shipment"
	OrderListStatusShipped           = "shipped"
	OrderListStatusDelivered         = "delivered"
	OrderPageSize                    = 25
	EventTypeProductionStatusChanged = "production_status_changed"
	EventTypeShippingStatusChanged   = "shipping_status_changed"
)

func NormalizeOrderListFilter(filter OrderListFilter) OrderListFilter {
	filter.Status = strings.TrimSpace(filter.Status)
	if !validOrderListStatus(filter.Status) {
		filter.Status = OrderListStatusAll
	}
	if filter.Page < 1 {
		filter.Page = 1
	}

	return filter
}

func OrderListOptions(activeStatus string) []OrderListOption {
	activeStatus = NormalizeOrderListFilter(OrderListFilter{Status: activeStatus, Page: 1}).Status
	statuses := []struct {
		value string
		label string
	}{
		{OrderListStatusAll, "Todos"},
		{OrderListStatusPendingPayment, "Aguardando pagamento"},
		{OrderListStatusWaitingProduction, "Aguardando producao"},
		{OrderListStatusInProduction, "Em producao"},
		{OrderListStatusWaitingShipment, "Aguardando envio"},
		{OrderListStatusPreparingShipment, "Preparando envio"},
		{OrderListStatusShipped, "Enviados"},
		{OrderListStatusDelivered, "Entregues"},
	}

	options := make([]OrderListOption, 0, len(statuses))
	for _, status := range statuses {
		option := OrderListOption{
			Value:  status.value,
			Label:  status.label,
			URL:    AdminOrdersURL(status.value, 1),
			Active: status.value == activeStatus,
		}
		options = append(options, option)
	}

	return options
}

func OrderListStatusLabel(status string) string {
	for _, option := range OrderListOptions(status) {
		if option.Value == status {
			return option.Label
		}
	}

	return "Todos"
}

func AdminOrdersURL(status string, page int) string {
	filter := NormalizeOrderListFilter(OrderListFilter{Status: status, Page: page})
	values := url.Values{}
	if filter.Status != OrderListStatusAll {
		values.Set("status", filter.Status)
	}
	if filter.Page > 1 {
		values.Set("page", strconv.Itoa(filter.Page))
	}
	if encoded := values.Encode(); encoded != "" {
		return "/admin/pedidos?" + encoded
	}

	return "/admin/pedidos"
}

func PrepareOrderListPage(filter OrderListFilter, items []OrderListItem) OrderListPage {
	filter = NormalizeOrderListFilter(filter)
	page := OrderListPage{
		Status:       filter.Status,
		StatusLabel:  OrderListStatusLabel(filter.Status),
		Page:         filter.Page,
		PreviousPage: filter.Page - 1,
		NextPage:     filter.Page + 1,
		HasPrevious:  filter.Page > 1,
		Options:      OrderListOptions(filter.Status),
	}

	if len(items) > OrderPageSize {
		page.HasNext = true
		items = items[:OrderPageSize]
	}
	for i := range items {
		PrepareOrderListItem(&items[i])
	}
	page.Orders = items

	return page
}

func PrepareOrderListItem(item *OrderListItem) {
	if item == nil {
		return
	}

	item.OrderNumberLabel = OrderNumberLabel(item.OrderNumber)
	item.OrderStatusLabel = OrderStatusLabel(item.OrderStatus)
	item.ProductionStatusLabel = ProductionStatusLabel(item.OrderStatus, item.ProductionStatus)
	item.ShippingStatusLabel = ShippingStatusLabel(item.OrderStatus, item.ProductionStatus, item.ShippingStatus)
	item.CreatedAtLabel = FormatOrderDate(item.CreatedAt)
	item.TotalBRL = products.FormatBRL(item.TotalCents)
	item.DetailURL = "/admin/pedidos/" + strings.ToLower(item.ID)
}

func PrepareOrderDetail(detail *OrderDetail) {
	if detail == nil {
		return
	}

	detail.OrderNumberLabel = OrderNumberLabel(detail.OrderNumber)
	detail.OrderStatusLabel = OrderStatusLabel(detail.OrderStatus)
	detail.ProductionStatusLabel = ProductionStatusLabel(detail.OrderStatus, detail.ProductionStatus)
	detail.ShippingStatusLabel = ShippingStatusLabel(detail.OrderStatus, detail.ProductionStatus, detail.ShippingStatus)
	detail.CreatedAtLabel = FormatOrderDate(detail.CreatedAt)
	detail.ProductsSubtotalBRL = products.FormatBRL(detail.ProductsSubtotalCents)
	detail.ShippingPriceBRL = products.FormatBRL(detail.ShippingPriceCents)
	detail.TotalBRL = products.FormatBRL(detail.TotalCents)
	if ValidUUID(detail.PublicTrackingID) {
		detail.TrackingURL = "/acompanhar/" + strings.ToLower(detail.PublicTrackingID)
	}
	detail.ProductionActions = ProductionActions(OrderStatusSnapshot{
		OrderStatus:      detail.OrderStatus,
		ProductionStatus: detail.ProductionStatus,
		ShippingStatus:   detail.ShippingStatus,
	})
	detail.ShippingActions = ShippingActions(OrderStatusSnapshot{
		OrderStatus:      detail.OrderStatus,
		ProductionStatus: detail.ProductionStatus,
		ShippingStatus:   detail.ShippingStatus,
	})
}

func ProductionActions(snapshot OrderStatusSnapshot) []OrderAction {
	if snapshot.OrderStatus != OrderStatusPaid || snapshot.ShippingStatus != ShippingStatusWaiting {
		return nil
	}

	switch snapshot.ProductionStatus {
	case ProductionStatusWaiting:
		return []OrderAction{{Status: ProductionStatusInProduction, Label: "Iniciar producao"}}
	case ProductionStatusInProduction:
		return []OrderAction{{Status: ProductionStatusCompleted, Label: "Concluir producao"}}
	default:
		return nil
	}
}

func ShippingActions(snapshot OrderStatusSnapshot) []OrderAction {
	if snapshot.OrderStatus != OrderStatusPaid || snapshot.ProductionStatus != ProductionStatusCompleted {
		return nil
	}

	switch snapshot.ShippingStatus {
	case ShippingStatusWaiting:
		return []OrderAction{{Status: ShippingStatusPreparing, Label: "Preparar envio"}}
	case ShippingStatusPreparing:
		return []OrderAction{{Status: ShippingStatusShipped, Label: "Marcar como enviado"}}
	case ShippingStatusShipped:
		return []OrderAction{{Status: ShippingStatusDelivered, Label: "Marcar como entregue"}}
	default:
		return nil
	}
}

func ValidateProductionTransition(snapshot OrderStatusSnapshot, targetStatus string) error {
	targetStatus = strings.TrimSpace(targetStatus)
	if targetStatus != ProductionStatusInProduction && targetStatus != ProductionStatusCompleted {
		return ErrInvalidTransition
	}
	if snapshot.ProductionStatus == targetStatus {
		return ErrTransitionConflict
	}
	for _, action := range ProductionActions(snapshot) {
		if action.Status == targetStatus {
			return nil
		}
	}

	return ErrInvalidTransition
}

func ValidateShippingTransition(snapshot OrderStatusSnapshot, targetStatus string) error {
	targetStatus = strings.TrimSpace(targetStatus)
	if targetStatus != ShippingStatusPreparing && targetStatus != ShippingStatusShipped && targetStatus != ShippingStatusDelivered {
		return ErrInvalidTransition
	}
	if snapshot.ShippingStatus == targetStatus {
		return ErrTransitionConflict
	}
	for _, action := range ShippingActions(snapshot) {
		if action.Status == targetStatus {
			return nil
		}
	}

	return ErrInvalidTransition
}

func OrderNumberLabel(orderNumber int64) string {
	return "#" + strconv.FormatInt(orderNumber, 10)
}

func OrderStatusLabel(status string) string {
	switch status {
	case OrderStatusPendingPayment:
		return "Aguardando pagamento"
	case OrderStatusPaid:
		return "Pagamento confirmado"
	default:
		return "Status indisponivel"
	}
}

func ProductionStatusLabel(orderStatus string, productionStatus string) string {
	if orderStatus == OrderStatusPendingPayment && productionStatus == ProductionStatusWaiting {
		return "Aguardando pagamento"
	}
	switch productionStatus {
	case ProductionStatusWaiting:
		return "Aguardando producao"
	case ProductionStatusInProduction:
		return "Em producao"
	case ProductionStatusCompleted:
		return "Producao concluida"
	default:
		return "Status indisponivel"
	}
}

func ShippingStatusLabel(orderStatus string, productionStatus string, shippingStatus string) string {
	switch shippingStatus {
	case ShippingStatusPreparing:
		return "Preparando envio"
	case ShippingStatusShipped:
		return "Enviado"
	case ShippingStatusDelivered:
		return "Entregue"
	}

	if orderStatus == OrderStatusPendingPayment {
		return "Aguardando pagamento"
	}
	switch productionStatus {
	case ProductionStatusWaiting:
		return "Aguardando producao"
	case ProductionStatusInProduction:
		return "Aguardando conclusao"
	case ProductionStatusCompleted:
		return "Aguardando envio"
	default:
		return "Status indisponivel"
	}
}

func PaymentStatusLabel(status string) string {
	switch status {
	case "pending":
		return "Pendente"
	case "paid":
		return "Pago"
	default:
		return "Nao iniciado"
	}
}

func EventTypeLabel(eventType string) string {
	switch eventType {
	case EventTypeProductionStatusChanged:
		return "Producao"
	case EventTypeShippingStatusChanged:
		return "Envio"
	default:
		return "Operacao"
	}
}

func EventStatusLabel(eventType string, status string) string {
	switch eventType {
	case EventTypeProductionStatusChanged:
		return ProductionStatusLabel(OrderStatusPaid, status)
	case EventTypeShippingStatusChanged:
		return ShippingStatusLabel(OrderStatusPaid, ProductionStatusCompleted, status)
	default:
		return status
	}
}

func FormatOrderDate(value time.Time) string {
	if value.IsZero() {
		return ""
	}

	return value.Format("02/01/2006 15:04")
}

func FormatPostalCode(postalCode string) string {
	digits := onlyDigits(postalCode)
	if len(digits) != 8 {
		return postalCode
	}

	return digits[:5] + "-" + digits[5:]
}

func FormatWeightG(weightG int64) string {
	if weightG <= 0 {
		return ""
	}

	return strconv.FormatInt(weightG, 10) + " g"
}

func FormatWeightMg(weightMg int64) string {
	if weightMg > 0 && weightMg%1000 == 0 {
		return strconv.FormatInt(weightMg/1000, 10) + " g"
	}

	return strconv.FormatInt(weightMg, 10) + " mg"
}

func FormatDimensionsMM(height int, width int, length int) string {
	if height <= 0 || width <= 0 || length <= 0 {
		return ""
	}

	return strconv.Itoa(length) + " x " + strconv.Itoa(width) + " x " + strconv.Itoa(height) + " mm"
}

func FormatInstallments(value *int) string {
	if value == nil || *value <= 0 {
		return ""
	}
	if *value == 1 {
		return "1 parcela"
	}

	return strconv.Itoa(*value) + " parcelas"
}

func onlyDigits(value string) string {
	var builder strings.Builder
	for _, char := range value {
		if char >= '0' && char <= '9' {
			builder.WriteRune(char)
		}
	}

	return builder.String()
}

func validOrderListStatus(status string) bool {
	switch status {
	case "", OrderListStatusAll, OrderListStatusPendingPayment, OrderListStatusWaitingProduction,
		OrderListStatusInProduction, OrderListStatusWaitingShipment, OrderListStatusPreparingShipment,
		OrderListStatusShipped, OrderListStatusDelivered:
		return true
	default:
		return false
	}
}
