package domain

import "testing"

func TestUser_HasRole(t *testing.T) {
	u := User{Roles: []Role{RoleUser}}

	if !u.HasRole(RoleUser) {
		t.Fatal("expected user to have RoleUser")
	}
	if u.HasRole(RoleAdmin) {
		t.Fatal("expected user not to have RoleAdmin")
	}
}
