package game

import "testing"

func TestDecideInitialAdminPlan(t *testing.T) {
	tests := []struct {
		name            string
		userFound       bool
		existingIsAdmin bool
		anyAdminsExist  bool
		want            initialAdminPlan
	}{
		{
			name:            "missing user => create",
			userFound:       false,
			existingIsAdmin: false,
			anyAdminsExist:  false,
			want:            planCreateNew,
		},
		{
			name:            "existing admin => ensure",
			userFound:       true,
			existingIsAdmin: true,
			anyAdminsExist:  false,
			want:            planEnsureExistingAdmin,
		},
		{
			name:            "existing non-admin + some admin exists => noop",
			userFound:       true,
			existingIsAdmin: false,
			anyAdminsExist:  true,
			want:            planNoopExistingNonAdmin,
		},
		{
			name:            "existing non-admin + no admins => promote",
			userFound:       true,
			existingIsAdmin: false,
			anyAdminsExist:  false,
			want:            planPromoteExisting,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got := decideInitialAdminPlan(tt.userFound, tt.existingIsAdmin, tt.anyAdminsExist)
			if got != tt.want {
				t.Fatalf("decideInitialAdminPlan()=%v want=%v", got, tt.want)
			}
		})
	}
}
