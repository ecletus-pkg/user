package user

import (
	"github.com/moisespsena/go-i18n-modular/i18nmod"
	"github.com/moisespsena/go-path-helpers"
)

var (
	PKG        = path_helpers.GetCalledDir()
	I18N_GROUP = i18nmod.PkgToGroup(PKG)
)

func I18n(key ...string) string {
	if len(key) > 0 && key[0] != "" {
		return I18N_GROUP + "." + key[0]
	}
	return I18N_GROUP
}
