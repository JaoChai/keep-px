package facebook

import (
	"crypto/sha256"
	"fmt"
	"strings"
)

// piiFields are fields that must be SHA-256 hashed per Meta CAPI requirements.
var piiFields = map[string]bool{
	"em": true, "ph": true, "fn": true, "ln": true,
	"ge": true, "db": true, "ct": true, "st": true,
	"zp": true, "country": true, "external_id": true,
}

// HashValue normalizes and SHA-256 hashes a single value per Meta CAPI requirements.
func HashValue(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	hash := sha256.Sum256([]byte(normalized))
	return fmt.Sprintf("%x", hash)
}

// HashUserData hashes PII fields in user_data per Meta CAPI requirements.
// Non-PII fields (client_ip_address, client_user_agent, fbc, fbp) are passed through unchanged.
func HashUserData(userData map[string]interface{}) map[string]interface{} {
	if userData == nil {
		return userData
	}

	result := make(map[string]interface{}, len(userData))
	for k, v := range userData {
		if piiFields[k] {
			if s, ok := v.(string); ok && s != "" {
				normalized := strings.ToLower(strings.TrimSpace(s))
				hash := sha256.Sum256([]byte(normalized))
				result[k] = fmt.Sprintf("%x", hash)
			} else {
				result[k] = v
			}
		} else {
			result[k] = v
		}
	}
	return result
}
