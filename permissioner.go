package user

import (
	"net/http"

	"github.com/moisespsena-go/i18n-modular/i18nmod"

	"github.com/ecletus/roles"
)

var roleUserManager = PKG + "@user_manager"

func RoleUserManager() string {
	return roleUserManager
}

func init() {
	RegisterRole(roleUserManager, i18ng+".roles.user_manager")
}

func EqualsRoleChecker(roleName string) roles.Checker {
	return func(req *http.Request, user interface{}) bool {
		if roleHaver, ok := user.(roles.RoleHaver); ok {
			return roleHaver.HasRole(roleName)
		}
		return false
	}
}

func RegisterRole(name, label string) {
	roles.Register(roles.NewDescriptor(name, EqualsRoleChecker(name), i18nmod.Cached(label)))
}
