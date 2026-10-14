package domain

// Role is persisted by name so that adding or reordering roles never changes
// the meaning of data already stored, and so that the zero value ("") is never
// a valid (let alone privileged) role.
type Role string

const (
	SuperAdmin     Role = "SuperAdmin"
	Admin          Role = "Admin"
	ContentManager Role = "ContentManager"
	API            Role = "API"
)

var allRoles = []Role{SuperAdmin, Admin, ContentManager, API}

func AllRoles() []Role {
	roles := make([]Role, len(allRoles))
	copy(roles, allRoles)
	return roles
}

func (r Role) IsValid() bool {
	for _, role := range allRoles {
		if r == role {
			return true
		}
	}
	return false
}

func (r Role) String() string {
	return string(r)
}
