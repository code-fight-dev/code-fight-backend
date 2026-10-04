package access

import (
	"github.com/google/uuid"
	"testing"
)

func TestTournamentPermissions(t *testing.T) {
	owner, other := uuid.New(), uuid.New()
	for _, test := range []struct {
		role           Role
		actor          uuid.UUID
		create, manage bool
	}{
		{User, owner, false, false}, {Organizer, owner, true, true},
		{Organizer, other, true, false}, {Admin, other, true, true},
		{Role("superadmin"), owner, false, false}, {Role(""), owner, false, false},
		{Admin, uuid.Nil, true, false},
	} {
		if got := CanCreateTournament(test.role); got != test.create {
			t.Errorf("create %q = %v", test.role, got)
		}
		if got := CanManageTournament(test.role, test.actor, owner); got != test.manage {
			t.Errorf("manage %q/%s = %v", test.role, test.actor, got)
		}
	}
}
