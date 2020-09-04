package gsuite_sso_name_id_finder

import (
	"strings"

	"github.com/moisespsena-go/i18n-modular/i18nmod"
	path_helpers "github.com/moisespsena-go/path-helpers"
)

var i18nGroup = i18nmod.PkgToGroup(path_helpers.GetCalledDir())

type err string

func (this err) Translate(ctx i18nmod.Context) string {
	return ctx.TT(i18nGroup + ".errors." + string(this)).Data(ctx.Value("gsuite_sso_domain")).Get()
}

func (this err) Error() string {
	return strings.ReplaceAll(string(this), "_", " ")
}

func (this err) Cause(err error) (errs i18nmod.Errors) {
	return append(errs, this, err)
}

const (
	ErrUserDoesNotHaveMailForDomain err = "user_does_not_have_mail_for_domain"
)
