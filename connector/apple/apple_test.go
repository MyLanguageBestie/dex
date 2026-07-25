package apple

import "testing"

// Apple does not send email_verified consistently: a live sign-in omitted it entirely, and it has
// been observed both as a real boolean and as the string "true". Reading it with a plain bool type
// assertion produced "missing email_verified claim" — a 500 at the very last step, after the learner
// had already authenticated with Apple. These pin all three shapes.
func TestAppleEmailVerified(t *testing.T) {
	tests := []struct {
		name  string
		claim interface{}
		want  bool
	}{
		{"boolean true", true, true},
		{"boolean false", false, false},
		{"string true, which Apple also sends", "true", true},
		{"string false", "false", false},
		// The case that broke a real sign-in. Apple only ever hands over an address it controls —
		// the account's verified address, or an @privaterelay alias it operates — so an absent
		// claim means verified, not unverified.
		{"absent", nil, true},
		{"some shape nobody expected", 42, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := appleEmailVerified(tt.claim); got != tt.want {
				t.Errorf("appleEmailVerified(%#v) = %v, want %v", tt.claim, got, tt.want)
			}
		})
	}
}
