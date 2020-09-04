package user

import (
	"github.com/moisespsena-go/i18n-modular/i18nmod"
	"github.com/moisespsena-go/path-helpers"
)

const AdminUserName = "admin"

var (
	PKG   = path_helpers.GetCalledDir()
	i18ng = i18nmod.PkgToGroup(PKG)
)

func I18n(key ...string) string {
	if len(key) > 0 && key[0] != "" {
		return i18ng + "." + key[0]
	}
	return i18ng
}
