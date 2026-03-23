package users

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadLegacyUsers(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "dump.json")

	content := `{
	  "rows": [
	    {
	      "doc": {
	        "_id": "department_1_dept-1",
	        "title": "Operations"
	      }
	    },
	    {
	      "doc": {
	        "_id": "user_1_user-a",
	        "firstName": "Алексей",
	        "lastName": "Смирнов",
	        "department": "dept-1",
	        "email": "Alex@example.com"
	      }
	    },
	    {
	      "doc": {
	        "_id": "user_1_user-b",
	        "firstName": "Alex",
	        "lastName": "Smith",
	        "department": "dept-1",
	        "email": "alex@example.com"
	      }
	    },
	    {
	      "doc": {
	        "_id": "user_1_user-c",
	        "firstName": "Alex",
	        "lastName": "Smith",
	        "department": "missing",
	        "email": ""
	      }
	    }
	  ]
	}`

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write dump: %v", err)
	}

	plan, err := loadLegacyImportPlan(path)
	if err != nil {
		t.Fatalf("loadLegacyImportPlan: %v", err)
	}

	users := plan.Users
	if len(users) != 3 {
		t.Fatalf("expected 3 users, got %d", len(users))
	}

	if users[0].Username != "smirnov.aleksey" {
		t.Fatalf("expected first username to use transliterated lastname.firstname format, got %q", users[0].Username)
	}

	if users[1].Username != "smith.alex" {
		t.Fatalf("expected second username to use lastname.firstname format, got %q", users[1].Username)
	}

	if users[2].Username != "smith.alex-2" {
		t.Fatalf("expected duplicate names to receive a deterministic suffix, got %q", users[2].Username)
	}

	if users[0].DepartmentTitle != "Operations" {
		t.Fatalf("expected department title to resolve, got %q", users[0].DepartmentTitle)
	}

	if users[2].DepartmentTitle != "" {
		t.Fatalf("expected missing department to stay empty, got %q", users[2].DepartmentTitle)
	}

	if got := plan.Departments["dept-1"]; got != "Operations" {
		t.Fatalf("expected department map to include imported title, got %q", got)
	}
}

func TestTrimLegacyPrefix(t *testing.T) {
	t.Parallel()

	if got := trimLegacyPrefix("user_1_abc-123"); got != "abc-123" {
		t.Fatalf("unexpected value %q", got)
	}

	if got := trimLegacyPrefix("plain-value"); got != "plain-value" {
		t.Fatalf("unexpected value %q", got)
	}
}

func TestLegacyUsernameBase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		firstName string
		lastName  string
		want      string
	}{
		{
			name:      "latin with accents",
			firstName: "José",
			lastName:  "Núñez",
			want:      "nunez.jose",
		},
		{
			name:      "cyrillic transliteration",
			firstName: "Мария",
			lastName:  "Ильина",
			want:      "ilina.mariya",
		},
		{
			name:      "hyphenated names",
			firstName: "Anne Marie",
			lastName:  "Smith-Jones",
			want:      "smith-jones.anne-marie",
		},
		{
			name:      "missing last name",
			firstName: "Alex",
			lastName:  "",
			want:      "alex",
		},
		{
			name:      "missing names",
			firstName: "",
			lastName:  "",
			want:      "user",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := legacyUsernameBase(tt.firstName, tt.lastName); got != tt.want {
				t.Fatalf("legacyUsernameBase(%q, %q) = %q, want %q", tt.firstName, tt.lastName, got, tt.want)
			}
		})
	}
}
