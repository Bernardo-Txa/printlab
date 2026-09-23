package shipping

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestSuperFreteContractCalculatorPackageOptIn(t *testing.T) {
	if os.Getenv("SUPERFRETE_CONTRACT_TEST") != "1" {
		t.Skip("set SUPERFRETE_CONTRACT_TEST=1 to run the real SuperFrete package contract test")
	}

	environment, token, origin, contactEmail, services, destination := superFreteContractEnv(t)
	client, err := NewSuperFreteClient(SuperFreteClientConfig{
		Environment:  environment,
		APIToken:     token,
		ContactEmail: contactEmail,
	})
	if err != nil {
		t.Fatalf("expected SuperFrete client configuration, got %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	quotes, err := client.Calculate(ctx, SuperFreteCalculatorRequest{
		FromPostalCode: origin,
		ToPostalCode:   destination,
		Services:       services,
		Package: &SuperFretePackage{
			WeightKG: 0.283,
			HeightCM: 10,
			WidthCM:  20,
			LengthCM: 20,
		},
	})
	if err != nil {
		t.Fatalf("expected SuperFrete package quote, got safe error %v", err)
	}
	if len(quotes) == 0 {
		t.Fatal("expected at least one valid final SuperFrete quote")
	}
}

func TestSuperFreteContractCalculatorProductsCompatibilityOptIn(t *testing.T) {
	if os.Getenv("SUPERFRETE_PRODUCTS_CONTRACT_TEST") != "1" {
		t.Skip("set SUPERFRETE_PRODUCTS_CONTRACT_TEST=1 to monitor the non-commercial products contract")
	}

	environment, token, origin, contactEmail, services, destination := superFreteContractEnv(t)
	client, err := NewSuperFreteClient(SuperFreteClientConfig{
		Environment:  environment,
		APIToken:     token,
		ContactEmail: contactEmail,
	})
	if err != nil {
		t.Fatalf("expected SuperFrete client configuration, got %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	quotes, err := client.Calculate(ctx, SuperFreteCalculatorRequest{
		FromPostalCode: origin,
		ToPostalCode:   destination,
		Services:       services,
		Products: []SuperFreteProduct{
			{Quantity: 1, WeightKG: 0.183, HeightCM: 10, WidthCM: 7, LengthCM: 7},
		},
	})
	if err != nil {
		t.Fatalf("expected SuperFrete products quote, got safe error %v", err)
	}
	if _, ok := firstReturnedPackage(quotes); !ok {
		t.Fatal("expected SuperFrete products quote to return package")
	}
}

func superFreteContractEnv(t *testing.T) (environment string, token string, origin string, contactEmail string, services string, destination string) {
	t.Helper()
	environment = strings.TrimSpace(os.Getenv("SUPERFRETE_ENV"))
	token = strings.TrimSpace(os.Getenv("SUPERFRETE_API_TOKEN"))
	origin = normalizeDigitsForContractTest(os.Getenv("SUPERFRETE_ORIGIN_POSTAL_CODE"))
	contactEmail = strings.TrimSpace(os.Getenv("SUPERFRETE_CONTACT_EMAIL"))
	services = strings.TrimSpace(os.Getenv("SUPERFRETE_SERVICES"))
	destination = normalizeDigitsForContractTest(os.Getenv("SUPERFRETE_TEST_DESTINATION_POSTAL_CODE"))
	if destination == "" {
		destination = origin
	}
	if environment == "" || token == "" || origin == "" || contactEmail == "" || services == "" || destination == "" {
		t.Fatal("SuperFrete contract test requires complete SUPERFRETE_* environment")
	}
	return environment, token, origin, contactEmail, services, destination
}

func normalizeDigitsForContractTest(value string) string {
	var digits strings.Builder
	for _, char := range value {
		if char >= '0' && char <= '9' {
			digits.WriteRune(char)
		}
	}
	return digits.String()
}
