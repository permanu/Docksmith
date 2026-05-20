package emit

import "testing"

func TestEmitDockerfileNilPlanReturnsEmpty(t *testing.T) {
	if got := EmitDockerfile(nil); got != "" {
		t.Fatalf("EmitDockerfile(nil) = %q, want empty string", got)
	}
}
