package theme

import "testing"

func TestCanonicalID(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"", Violet},
		{"VIOLET", Violet},
		{"blue", Sky},
		{"bluesky", Sky},
		{"warm", Orange},
		{"grey", Neutral},
		{"mint", Green},
	} {
		got, err := CanonicalID(tc.in)
		if err != nil {
			t.Fatalf("%q: %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("%q: got %q want %q", tc.in, got, tc.want)
		}
	}
	if _, err := CanonicalID("nope"); err == nil {
		t.Fatal("expected error")
	}
}
