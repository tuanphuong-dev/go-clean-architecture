package utils

import "unicode/utf8"

func MaskFullName(name string) string {
	if len(name) == 0 {
		return ""
	}
	return string(name[0]) + "***"
}

// MaskEmail masks the email address by replacing the local part with asterisks
// Example: abcd@domain.com -> a***@domain.com
func MaskEmail(email *string) string {
	if email == nil || *email == "" {
		return ""
	}
	at := -1
	for i, c := range *email {
		if c == '@' {
			at = i
			break
		}
	}
	if at <= 1 {
		return "***" + (*email)[at:]
	}
	return (*email)[:1] + "***" + (*email)[at:]
}

// MaskPhone masks a phone number, keeping country prefix (e.g., +84)
// and last 3 digits visible, masking the middle part with '*'.
func MaskPhone(phone string) string {
	if phone == "" {
		return ""
	}

	if utf8.RuneCountInString(phone) <= 8 {
		return phone
	}

	// Keep first 5 characters (country code or area code)
	// If phone starts with +, we keep +84 or similar
	// Otherwise, we keep the first 5 digits
	prefixLen := 5
	if len(phone) > 5 && phone[0] == '+' {
		// Keep +84
		prefixLen = 5
	} else {
		// Keep first 5 characters
		prefixLen = 5
	}

	// Keep last 3 digits
	suffixLen := 3

	// If phone is shorter than prefix + suffix, return as is
	if len(phone) <= prefixLen+suffixLen {
		return phone
	}

	prefix := phone[:prefixLen]
	suffix := phone[len(phone)-suffixLen:]
	masked := ""
	maskCount := len(phone) - prefixLen - suffixLen
	for range maskCount {
		masked += "*"
	}
	return prefix + masked + suffix
}
