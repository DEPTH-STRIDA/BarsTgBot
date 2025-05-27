package masker

import (
	"reflect"
	"testing"

	"go.uber.org/zap/zaptest"
)

type Inner struct {
	Secret string `masked:"true"`
	Plain  string
}

type Config struct {
	Name  string
	Inner Inner
}

type ConfigMasked struct {
	Password string `masked:"true"`
	Token    string `masked:"true"`
	Email    string
}

func TestMaskSensitiveData(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"secret", "s****t"},
		{"ab", "****"},
		{"a", "****"},
		{"", "****"},
	}
	for _, tt := range tests {
		got := maskSensitiveData(tt.in)
		if got != tt.want {
			t.Errorf("maskSensitiveData(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestMaskStructFields(t *testing.T) {
	// Структуру для конфигурации
	cfg := Config{
		Name: "test", // не будет замаскировано
		Inner: Inner{
			Secret: "mypassword", // будет замаскировано
			Plain:  "visible",    // не будет замаскировано
		},
	}
	v := reflect.ValueOf(cfg)
	tp := reflect.TypeOf(cfg)
	got := maskStructFields(v, tp)

	inner, ok := got["Inner"].(map[string]interface{})
	if !ok {
		t.Fatal("Inner field not mapped correctly")
	}
	if inner["Secret"] != "m****d" {
		t.Errorf("Secret masked incorrectly: got %v", inner["Secret"])
	}
	if inner["Plain"] != "visible" {
		t.Errorf("Plain field incorrect: got %v", inner["Plain"])
	}
	if got["Name"] != "test" {
		t.Errorf("Name field incorrect: got %v", got["Name"])
	}
}

func TestMaskStructFields_MaskedFields(t *testing.T) {
	cfg := ConfigMasked{
		Password: "supersecret",
		Token:    "tok12345",
		Email:    "user@example.com",
	}
	v := reflect.ValueOf(cfg)
	tp := reflect.TypeOf(cfg)
	got := maskStructFields(v, tp)
	if got["Password"] != "s****t" {
		t.Errorf("Password masked incorrectly: got %v", got["Password"])
	}
	if got["Token"] != "t****5" {
		t.Errorf("Token masked incorrectly: got %v", got["Token"])
	}
	if got["Email"] != "user@example.com" {
		t.Errorf("Email field incorrect: got %v", got["Email"])
	}
}

func TestLogConfigs_Success(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cfg := &ConfigMasked{
		Password: "supersecret",
		Token:    "tok12345",
		Email:    "user@example.com",
	}
	if err := LogConfigs(logger, cfg); err != nil {
		t.Errorf("LogConfigs returned error: %v", err)
	}
}

func TestLogConfigs_NotPointer(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cfg := ConfigMasked{
		Password: "supersecret",
		Token:    "tok12345",
		Email:    "user@example.com",
	}
	if err := LogConfigs(logger, cfg); err == nil {
		t.Error("expected error when passing non-pointer, but got nil")
	}
}
