package customerauth

import "testing"

func TestValidateSignup(t *testing.T) {
	input, errors := ValidateSignup(" Maria ", " MARIA@EXAMPLE.COM ", "senha-segura", "senha-segura")
	if errors.Any() {
		t.Fatalf("expected valid signup, got %#v", errors)
	}
	if input.Name != "Maria" || input.Email != "maria@example.com" || input.Password != "senha-segura" {
		t.Fatalf("expected normalized input, got %#v", input)
	}
}

func TestValidateSignupPasswordConfirmation(t *testing.T) {
	_, errors := ValidateSignup("Maria", "maria@example.com", "senha-segura", "outra-senha")
	if errors.Message("confirm_password") != "As senhas não coincidem." {
		t.Fatalf("expected password confirmation error, got %#v", errors)
	}
}

func TestSafeRedirectPath(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    string
		wantErr bool
	}{
		{name: "empty", value: "", want: "/conta"},
		{name: "local", value: "/conta?aba=perfil", want: "/conta?aba=perfil"},
		{name: "external", value: "https://evil.example", wantErr: true},
		{name: "protocol relative", value: "//evil.example", wantErr: true},
		{name: "admin", value: "/admin", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SafeRedirectPath(tt.value, "/conta")
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %q", got)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Fatalf("expected %q nil, got %q %v", tt.want, got, err)
			}
		})
	}
}
