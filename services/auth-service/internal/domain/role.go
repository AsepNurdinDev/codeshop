package domain

// Role represents a user's authorization role.
//
// Per docs/architecture/DATABASE_DESIGN.md §2, full RBAC (a separate Role
// entity with many-to-many assignment) is deliberately deferred: for the
// MVP a simple enum field on User is sufficient since only two roles exist.
type Role string

const (
	// RoleBuyer is the default role for any newly registered user.
	RoleBuyer Role = "buyer"

	// RoleAdmin grants access to administrative operations (e.g. managing
	// the product catalog) in other services. Auth Service itself does not
	// enforce catalog permissions, but issues the role claim that other
	// services trust.
	RoleAdmin Role = "admin"
)

// Valid reports whether r is one of the known roles.
func (r Role) Valid() bool {
	switch r {
	case RoleBuyer, RoleAdmin:
		return true
	default:
		return false
	}
}

func (r Role) String() string {
	return string(r)
}
