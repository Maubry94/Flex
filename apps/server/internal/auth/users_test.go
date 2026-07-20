package auth

import (
	"context"
	"errors"
	"testing"

	"flex.local/server/internal/database"
)

func TestManageUsersAndProtectLastAdministrator(t *testing.T) {
	db, err := database.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	service := NewService(db)
	service.iterations = 100_000
	ctx := context.Background()
	adminSession, err := service.Setup(ctx, "admin", "correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	member, err := service.CreateUser(ctx, "member", "another secure password", "user")
	if err != nil {
		t.Fatal(err)
	}
	users, err := service.ListUsers(ctx)
	if err != nil || len(users) != 2 {
		t.Fatalf("unexpected users: %#v, %v", users, err)
	}
	if _, err := service.UpdateUser(ctx, adminSession.User.ID, UserInput{Username: "admin", Role: "user", Active: true}); !errors.Is(err, ErrLastAdmin) {
		t.Fatalf("expected ErrLastAdmin, got %v", err)
	}
	updated, err := service.UpdateUser(ctx, member.ID, UserInput{Username: "viewer", Role: "admin", Active: true})
	if err != nil || updated.Role != "admin" || updated.Username != "viewer" {
		t.Fatalf("unexpected updated user: %#v, %v", updated, err)
	}
	if err := service.DeleteUser(ctx, adminSession.User.ID); err != nil {
		t.Fatalf("delete first administrator: %v", err)
	}
}

func TestResetPasswordRevokesSessions(t *testing.T) {
	db, err := database.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	service := NewService(db)
	service.iterations = 100_000
	ctx := context.Background()
	if _, err := service.Setup(ctx, "admin", "correct horse battery staple"); err != nil {
		t.Fatal(err)
	}
	member, err := service.CreateUser(ctx, "member", "another secure password", "user")
	if err != nil {
		t.Fatal(err)
	}
	session, err := service.Login(ctx, "member", "another secure password")
	if err != nil {
		t.Fatal(err)
	}
	if err := service.ResetPassword(ctx, member.ID, "brand new secure password"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Authenticate(ctx, session.Token); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("expected old session to be revoked, got %v", err)
	}
}
