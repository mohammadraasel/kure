package importt

import (
	"testing"

	cmdutil "github.com/GGP1/kure/cmd"
	"github.com/GGP1/kure/db/entry"
	"github.com/GGP1/kure/pb"
)

func TestImport(t *testing.T) {
	db := cmdutil.SetContext(t, "../../db/testdata/database")
	defer db.Close()

	cases := []struct {
		manager  string
		path     string
		name     string
		expected *pb.Entry
	}{
		{
			manager: "Keepass",
			path:    "testdata/test_keepass",
			name:    "keepass",
			expected: &pb.Entry{
				Username: "test@keepass.com",
				Password: "keepass123",
				URL:      "https://keepass.info/",
				Notes:    "Notes",
				Expires:  "Never",
			},
		},
		{
			manager: "1password",
			path:    "testdata/test_1password.csv",
			name:    "1password",
			expected: &pb.Entry{
				Username: "test@1password.com",
				Password: "1password123",
				URL:      "https://1password.com/",
				Notes:    "Notes. Member number: 1234. Recovery Codes: The Shire",
				Expires:  "Never",
			},
		},
		{
			manager: "Lastpass",
			path:    "testdata/test_lastpass.csv",
			// Kure will by default join folders with the entry names
			name: "test/lastpass",
			expected: &pb.Entry{
				Username: "test@lastpass.com",
				Password: "lastpass123",
				URL:      "https://lastpass.com/",
				Notes:    "Notes",
				Expires:  "Never",
			},
		},
		{
			manager: "Bitwarden",
			path:    "testdata/test_bitwarden.csv",
			// Kure will by default join folders with the entry names
			name: "test/bitwarden",
			expected: &pb.Entry{
				Username: "test@bitwarden.com",
				Password: "bitwarden123",
				URL:      "https://bitwarden.com/",
				Notes:    "Notes",
				Expires:  "Never",
			},
		},
	}

	cmd := NewCmd(db)

	for _, tc := range cases {
		t.Run(tc.manager, func(t *testing.T) {
			cmd.Flags().Set("path", tc.path)
			args := []string{tc.manager}

			if err := cmd.RunE(cmd, args); err != nil {
				t.Fatalf("Failed importing entries: %v", err)
			}

			_, got, err := entry.Get(db, tc.name)
			if err != nil {
				t.Fatalf("Failed listing entry: %v", err)
			}

			compareEntries(t, got, tc.expected)
		})
	}
}

func TestInvalidImport(t *testing.T) {
	db := cmdutil.SetContext(t, "../../db/testdata/database")
	defer db.Close()

	cases := []struct {
		desc    string
		manager string
		path    string
	}{
		{
			desc:    "Invalid name",
			manager: "",
			path:    "testdata/test_keepass.csv",
		},
		{
			desc:    "Invalid path",
			manager: "keepass",
			path:    "",
		},
		{
			desc:    "Failed reading file",
			manager: "1password",
			path:    "test.csv",
		},
		{
			desc:    "Unsupported manager",
			manager: "unsupported",
			path:    "testdata/test_keepass.csv",
		},
	}

	cmd := NewCmd(db)

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			cmd.Flags().Set("path", tc.path)
			args := []string{tc.manager}

			if err := cmd.RunE(cmd, args); err == nil {
				t.Error("Expected an error but got nil")
			}
		})
	}
}

func TestPostRun(t *testing.T) {
	db := cmdutil.SetContext(t, "../../db/testdata/database")
	defer db.Close()

	cmd := NewCmd(db)
	cmd.PostRun(nil, nil)
}

func compareEntries(t *testing.T, got, expected *pb.Entry) {
	if got.Username != expected.Username {
		t.Errorf("Expected %s, got %s", expected.Username, got.Username)
	}

	if got.Password != expected.Password {
		t.Errorf("Expected %s, got %s", expected.Password, got.Password)
	}

	if got.URL != expected.URL {
		t.Errorf("Expected %s, got %s", expected.URL, got.URL)
	}

	if got.Notes != expected.Notes {
		t.Errorf("Expected %s, got %s", expected.Notes, got.Notes)
	}

	if got.Expires != expected.Expires {
		t.Errorf("Expected %s, got %s", expected.Expires, got.Expires)
	}
}
