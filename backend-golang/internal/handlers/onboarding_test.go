package handlers

import "testing"

func TestNormalizeOnboardingAuthMethod(t *testing.T) {
	t.Parallel()

	if got := normalizeOnboardingAuthMethod("  PaSsWoRd  "); got != onboardingAuthMethodPassword {
		t.Fatalf("expected normalized auth method %q, got %q", onboardingAuthMethodPassword, got)
	}
}

func TestValidateOnboardingAuthMethod(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		authMethod string
		wantErr    string
	}{
		{
			name:       "password is allowed",
			authMethod: onboardingAuthMethodPassword,
		},
		{
			name:       "sso is rejected until implemented",
			authMethod: "sso",
			wantErr:    "SSO onboarding is not available yet. Use password authentication for now",
		},
		{
			name:       "invalid values are rejected",
			authMethod: "magic-link",
			wantErr:    "auth_method must be 'password'",
		},
		{
			name:       "blank is rejected",
			authMethod: "",
			wantErr:    "auth_method must be 'password'",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateOnboardingAuthMethod(tt.authMethod)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error %q, got nil", tt.wantErr)
			}
			if err.Error() != tt.wantErr {
				t.Fatalf("expected error %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

func TestNormalizeOnboardingTeamMembers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		adminEmail  string
		teamMembers []onboardingTeamMemberRequest
		wantErr     string
		wantCount   int
	}{
		{
			name:       "normalizes valid team members",
			adminEmail: "owner@example.com",
			teamMembers: []onboardingTeamMemberRequest{
				{Email: " HR.Manager@example.com ", Role: "HR"},
				{Email: "admin2@example.com", Role: "org_admin"},
			},
			wantCount: 2,
		},
		{
			name:       "rejects primary admin email reuse",
			adminEmail: "owner@example.com",
			teamMembers: []onboardingTeamMemberRequest{
				{Email: "owner@example.com", Role: "hr"},
			},
			wantErr: "team member emails must be unique and cannot match the primary admin email",
		},
		{
			name:       "rejects duplicate team emails",
			adminEmail: "owner@example.com",
			teamMembers: []onboardingTeamMemberRequest{
				{Email: "dup@example.com", Role: "hr"},
				{Email: " DUP@example.com ", Role: "org_admin"},
			},
			wantErr: "team member emails must be unique and cannot match the primary admin email",
		},
		{
			name:       "rejects department manager role",
			adminEmail: "owner@example.com",
			teamMembers: []onboardingTeamMemberRequest{
				{Email: "manager@example.com", Role: "dept_manager"},
			},
			wantErr: "department managers can be assigned after departments are created",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := normalizeOnboardingTeamMembers(tt.teamMembers, tt.adminEmail)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if len(got) != tt.wantCount {
					t.Fatalf("expected %d team members, got %d", tt.wantCount, len(got))
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error %q, got nil", tt.wantErr)
			}
			if err.Error() != tt.wantErr {
				t.Fatalf("expected error %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

func TestValidateOnboardingPlan(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		planTier           string
		estimatedEmployees int
		wantErr            string
	}{
		{
			name:               "starter within limit",
			planTier:           "starter",
			estimatedEmployees: 25,
		},
		{
			name:               "professional within limit",
			planTier:           "professional",
			estimatedEmployees: 200,
		},
		{
			name:               "enterprise unlimited",
			planTier:           "enterprise",
			estimatedEmployees: 5000,
		},
		{
			name:               "invalid plan rejected",
			planTier:           "free",
			estimatedEmployees: 10,
			wantErr:            "plan_tier must be 'starter', 'professional', or 'enterprise'",
		},
		{
			name:               "non positive employees rejected",
			planTier:           "starter",
			estimatedEmployees: 0,
			wantErr:            "estimated_employees must be greater than zero",
		},
		{
			name:               "starter over capacity rejected",
			planTier:           "starter",
			estimatedEmployees: 26,
			wantErr:            "starter supports up to 25 employees. Choose a higher plan or lower the estimated employee count",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := validateOnboardingPlan(tt.planTier, tt.estimatedEmployees)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error %q, got nil", tt.wantErr)
			}
			if err.Error() != tt.wantErr {
				t.Fatalf("expected error %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}
