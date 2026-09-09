package aws

import "testing"

// TestNewSessionRequiresRegionKeyIdSecret guards the remaining required-field
// validation in newSession.
func TestNewSessionRequiresRegionKeyIdSecret(t *testing.T) {
	cases := []struct {
		name   string
		params Params
		want   string
	}{
		{"missing region", Params{KeyId: "id", Secret: "secret"}, MSG_REGION_REQUIRED},
		{"missing keyId", Params{Region: "us-east-1", Secret: "secret"}, MSG_KEY_ID_REQUIRED},
		{"missing secret", Params{Region: "us-east-1", KeyId: "id"}, MSG_SECRET_REQUIRED},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := newSession(c.params)
			if err == nil || err.Error() != c.want {
				t.Fatalf("newSession(%+v) error = %v, want %q", c.params, err, c.want)
			}
		})
	}
}

// TestNewSessionDoesNotRequireToken guards against newSession rejecting the
// most common AWS credential shape: a permanent IAM access key/secret pair
// with no session token (session tokens only exist for temporary STS
// credentials). Before the fix, this always failed with MSG_TOKEN_REQUIRED.
func TestNewSessionDoesNotRequireToken(t *testing.T) {
	sess, err := newSession(Params{Region: "us-east-1", KeyId: "id", Secret: "secret"})
	if err != nil {
		t.Fatalf("newSession with empty token failed: %v", err)
	}
	if sess == nil {
		t.Fatal("expected a non-nil session")
	}
}

// TestNewSessionWithToken guards the temporary-credentials (STS) path still
// works with a token set.
func TestNewSessionWithToken(t *testing.T) {
	sess, err := newSession(Params{Region: "us-east-1", KeyId: "id", Secret: "secret", Token: "tok"})
	if err != nil {
		t.Fatalf("newSession with token failed: %v", err)
	}
	if sess == nil {
		t.Fatal("expected a non-nil session")
	}
}
