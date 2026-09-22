package components

import "strconv"

func cartDisplayCount(count int) string {
	if count > 99 {
		return "99+"
	}
	return strconv.Itoa(count)
}

func cartAriaLabel(count int) string {
	if count == 1 {
		return "Carrinho, 1 unidade"
	}
	if count > 1 {
		return "Carrinho, " + strconv.Itoa(count) + " unidades"
	}
	return "Carrinho"
}
