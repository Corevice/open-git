package ssh

import "testing"

func TestParseGitSSHCommand_UploadPack(t *testing.T) {
	tests := []struct {
		name       string
		command    string
		ownerLogin string
		repoName   string
	}{
		{
			name:       "quoted path with leading slash",
			command:    "git-upload-pack '/owner/repo.git'",
			ownerLogin: "owner",
			repoName:   "repo",
		},
		{
			name:       "quoted path without leading slash",
			command:    "git-upload-pack 'owner/my-repo.git'",
			ownerLogin: "owner",
			repoName:   "my-repo",
		},
		{
			name:       "unquoted path",
			command:    "git-upload-pack owner/repo.git",
			ownerLogin: "owner",
			repoName:   "repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := parseGitSSHCommand(tt.command)
			if err != nil {
				t.Fatalf("parseGitSSHCommand: %v", err)
			}
			if parsed.packType != "upload-pack" {
				t.Fatalf("packType = %q, want upload-pack", parsed.packType)
			}
			if parsed.ownerLogin != tt.ownerLogin {
				t.Fatalf("ownerLogin = %q, want %q", parsed.ownerLogin, tt.ownerLogin)
			}
			if parsed.repoName != tt.repoName {
				t.Fatalf("repoName = %q, want %q", parsed.repoName, tt.repoName)
			}
		})
	}
}

func TestParseGitSSHCommand_ReceivePack(t *testing.T) {
	tests := []struct {
		name       string
		command    string
		ownerLogin string
		repoName   string
	}{
		{
			name:       "quoted path with leading slash",
			command:    "git-receive-pack '/org/project.git'",
			ownerLogin: "org",
			repoName:   "project",
		},
		{
			name:       "quoted path without git suffix",
			command:    "git-receive-pack '/user/app'",
			ownerLogin: "user",
			repoName:   "app",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := parseGitSSHCommand(tt.command)
			if err != nil {
				t.Fatalf("parseGitSSHCommand: %v", err)
			}
			if parsed.packType != "receive-pack" {
				t.Fatalf("packType = %q, want receive-pack", parsed.packType)
			}
			if parsed.ownerLogin != tt.ownerLogin {
				t.Fatalf("ownerLogin = %q, want %q", parsed.ownerLogin, tt.ownerLogin)
			}
			if parsed.repoName != tt.repoName {
				t.Fatalf("repoName = %q, want %q", parsed.repoName, tt.repoName)
			}
		})
	}
}

func TestParseGitSSHCommand_InvalidCommand(t *testing.T) {
	invalid := []string{
		"",
		"git-upload-pack",
		"git-upload-pack '/only-one-segment.git'",
		"git-fetch '/owner/repo.git'",
		"upload-pack '/owner/repo.git'",
	}

	for _, command := range invalid {
		t.Run(command, func(t *testing.T) {
			_, err := parseGitSSHCommand(command)
			if err == nil {
				t.Fatalf("expected error for command %q", command)
			}
		})
	}
}
