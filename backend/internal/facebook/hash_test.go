package facebook

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHashUserData(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		checkKey string
		wantHash bool
		wantVal  string
	}{
		{
			name:     "hashes email",
			input:    map[string]interface{}{"em": "Test@Example.com"},
			checkKey: "em",
			wantHash: true,
		},
		{
			name:     "hashes phone",
			input:    map[string]interface{}{"ph": " 1234567890 "},
			checkKey: "ph",
			wantHash: true,
		},
		{
			name:     "passes through client_ip_address",
			input:    map[string]interface{}{"client_ip_address": "1.2.3.4"},
			checkKey: "client_ip_address",
			wantHash: false,
			wantVal:  "1.2.3.4",
		},
		{
			name:     "passes through client_user_agent",
			input:    map[string]interface{}{"client_user_agent": "Mozilla/5.0"},
			checkKey: "client_user_agent",
			wantHash: false,
			wantVal:  "Mozilla/5.0",
		},
		{
			name:     "passes through fbc",
			input:    map[string]interface{}{"fbc": "fb.1.123.abc"},
			checkKey: "fbc",
			wantHash: false,
			wantVal:  "fb.1.123.abc",
		},
		{
			name:     "passes through fbp",
			input:    map[string]interface{}{"fbp": "fb.1.456.def"},
			checkKey: "fbp",
			wantHash: false,
			wantVal:  "fb.1.456.def",
		},
		{
			name:     "nil input returns nil",
			input:    nil,
			checkKey: "",
			wantHash: false,
		},
		{
			name:     "empty string PII not hashed",
			input:    map[string]interface{}{"em": ""},
			checkKey: "em",
			wantHash: false,
			wantVal:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HashUserData(tt.input)
			if tt.input == nil {
				assert.Nil(t, result)
				return
			}
			val := result[tt.checkKey]
			if tt.wantHash {
				s, ok := val.(string)
				assert.True(t, ok)
				assert.Len(t, s, 64) // SHA-256 hex = 64 chars
				assert.NotEqual(t, tt.input[tt.checkKey], s)
			} else if tt.wantVal != "" {
				assert.Equal(t, tt.wantVal, val)
			}
		})
	}
}

func TestHashUserData_Normalization(t *testing.T) {
	r1 := HashUserData(map[string]interface{}{"em": "Test@Example.com"})
	r2 := HashUserData(map[string]interface{}{"em": "test@example.com"})
	r3 := HashUserData(map[string]interface{}{"em": " Test@Example.com "})
	assert.Equal(t, r1["em"], r2["em"])
	assert.Equal(t, r1["em"], r3["em"])
}
