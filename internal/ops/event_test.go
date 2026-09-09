package ops

import "testing"

func TestNewEventDTOSplitsActionType(t *testing.T) {
	tests := []struct {
		name        string
		actionType  string
		flagStatus  string
		wantMethod  string
		wantPath    string
		wantBlocked bool
	}{
		{
			name:       "a normal method and path split on the space",
			actionType: "GET /api/accounts",
			flagStatus: FlagAllowed,
			wantMethod: "GET",
			wantPath:   "/api/accounts",
		},
		{
			name:       "no space leaves the whole value as the path",
			actionType: "/api/accounts",
			flagStatus: FlagAllowed,
			wantMethod: "",
			wantPath:   "/api/accounts",
		},
		{
			name:       "only the first space splits, so a query with spaces survives",
			actionType: "GET /api/accounts?q=two words",
			flagStatus: FlagAllowed,
			wantMethod: "GET",
			wantPath:   "/api/accounts?q=two words",
		},
		{
			name:       "empty stays empty",
			actionType: "",
			flagStatus: FlagAllowed,
			wantMethod: "",
			wantPath:   "",
		},
		{
			name:        "Blocked is derived from the flag, not stored separately",
			actionType:  "GET /api/accounts",
			flagStatus:  FlagBlocked,
			wantMethod:  "GET",
			wantPath:    "/api/accounts",
			wantBlocked: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewEventDTO(SecurityEvent{ActionType: tt.actionType, FlagStatus: tt.flagStatus})

			if got.Method != tt.wantMethod {
				t.Errorf("Method = %q, want %q", got.Method, tt.wantMethod)
			}
			if got.Path != tt.wantPath {
				t.Errorf("Path = %q, want %q", got.Path, tt.wantPath)
			}
			if got.Blocked != tt.wantBlocked {
				t.Errorf("Blocked = %v, want %v", got.Blocked, tt.wantBlocked)
			}
			if got.ActionType != tt.actionType {
				t.Errorf("ActionType = %q, want it preserved as %q", got.ActionType, tt.actionType)
			}
		})
	}
}
