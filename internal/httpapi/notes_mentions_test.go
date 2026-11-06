package httpapi_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/Aurorainic/cinderfish/internal/note"
)

// TestNotesCreateMentionLimit pins Misskey's default role policy
// mentionLimit (20): a note that addresses more accounts is refused before it
// is stored, while the twentieth mention still passes.
func TestNotesCreateMentionLimit(t *testing.T) {
	// Mode none keeps the accepted note's delivery from resolving the
	// fake hosts over the network.
	f := newFederationServer(t, "none")
	token, _ := f.registerUser(t, "alice")

	mentions := func(n int) string {
		var b strings.Builder
		for i := 0; i < n; i++ {
			fmt.Fprintf(&b, "@user%d@remote.example ", i)
		}
		return strings.TrimSpace(b.String())
	}

	resp, out := postWithToken(t, f.ts, "/api/notes/create", token, map[string]any{
		"text": mentions(note.MaxMentions),
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("20 mentions: status %d body %v", resp.StatusCode, out)
	}

	resp, out = postWithToken(t, f.ts, "/api/notes/create", token, map[string]any{
		"text": mentions(note.MaxMentions + 1),
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("21 mentions: status %d body %v", resp.StatusCode, out)
	}
	e, _ := out["error"].(map[string]any)
	if e["code"] != "TOO_MANY_MENTIONS" {
		t.Errorf("code = %v, want TOO_MANY_MENTIONS", e["code"])
	}

	// The refused note must not have been stored.
	if ids := timelineIDs(t, f.ts, "/api/notes/timeline", token, nil); len(ids) != 1 {
		t.Errorf("timeline has %d notes, want 1", len(ids))
	}
}
