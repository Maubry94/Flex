package auth

import (
	"context"
	"errors"
	"testing"

	"flex.local/server/internal/database"
)

func TestSetupLoginAuthenticateAndLogout(t *testing.T) {
	db, err := database.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	service := NewService(db)
	service.iterations = 100_000
	ctx := context.Background()

	configured, err := service.Configured(ctx)
	if err != nil || configured {
		t.Fatalf("unexpected initial configuration: configured=%v error=%v", configured, err)
	}
	setupSession, err := service.Setup(ctx, "admin", "correct horse battery staple")
	if err != nil {
		t.Fatalf("Setup() returned an error: %v", err)
	}
	if setupSession.User.Role != "admin" || setupSession.Token == "" {
		t.Fatalf("unexpected setup session: %#v", setupSession)
	}
	if _, err := service.Setup(ctx, "other", "correct horse battery staple"); !errors.Is(err, ErrAlreadyConfigured) {
		t.Fatalf("expected ErrAlreadyConfigured, got %v", err)
	}
	if _, err := service.Login(ctx, "admin", "incorrect password"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
	loginSession, err := service.Login(ctx, "ADMIN", "correct horse battery staple")
	if err != nil {
		t.Fatalf("Login() returned an error: %v", err)
	}
	user, err := service.Authenticate(ctx, loginSession.Token)
	if err != nil || user.Username != "admin" {
		t.Fatalf("unexpected authenticated user: %#v, %v", user, err)
	}
	updatedUser, err := service.UpdateProfile(ctx, user.ID, "renamed-admin")
	if err != nil || updatedUser.Username != "renamed-admin" {
		t.Fatalf("unexpected updated profile: %#v, %v", updatedUser, err)
	}
	changedSession, err := service.ChangePassword(ctx, user.ID, "correct horse battery staple", "a different secure password")
	if err != nil {
		t.Fatalf("ChangePassword() returned an error: %v", err)
	}
	if _, err := service.Authenticate(ctx, loginSession.Token); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("old session should be revoked, got %v", err)
	}
	if _, err := service.Login(ctx, "renamed-admin", "correct horse battery staple"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("old password should be rejected, got %v", err)
	}
	if _, err := service.Authenticate(ctx, changedSession.Token); err != nil {
		t.Fatalf("replacement session should be valid: %v", err)
	}
	if err := service.Logout(ctx, changedSession.Token); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Authenticate(ctx, changedSession.Token); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("expected logged out session to be rejected, got %v", err)
	}
}

func TestSetupValidatesAdministratorCredentials(t *testing.T) {
	db, err := database.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	service := NewService(db)
	service.iterations = 100_000
	if _, err := service.Setup(context.Background(), "bad name", "too short"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}
