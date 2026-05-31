package leaderboard

import (
	"testing"

	"github.com/google/uuid"
)

func TestClonePageDeepCopiesSlicesAndViewerRank(t *testing.T) {
	viewerRank := &Entry{Rank: 10, UserID: uuid.New(), Username: "viewer"}
	page := Page{
		Items:      []Entry{{Rank: 1, UserID: uuid.New(), Username: "leader"}},
		ViewerRank: viewerRank,
	}

	cloned := clonePage(page)
	cloned.Items[0].Username = "changed"
	cloned.ViewerRank.Username = "changed-viewer"

	if page.Items[0].Username != "leader" {
		t.Fatalf("original item mutated: %#v", page.Items[0])
	}
	if page.ViewerRank.Username != "viewer" {
		t.Fatalf("original viewer rank mutated: %#v", page.ViewerRank)
	}
}
