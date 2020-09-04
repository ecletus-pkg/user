package gsuite_sso_name_id_finder

import (
	"github.com/ecletus-pkg/user"
	"github.com/moisespsena-go/xsaml/samlidp/drivers/gsuite"
	"github.com/moisespsena/go-ecletus-samlidp"
	"strings"
)


// finder this method is not valid for multiple domains in gsuinte account
var finder = ect_samlidp.NewNameIDFinder(func(ctx *ect_samlidp.AuthContext) (nameID string, err error) {
	if u, ok := ctx.AuthUser.(*user.User); ok {
		if domain := gsuite.GetDomain(ctx.AuthRequest); domain != "" {
			defer ctx.LocalContext.BackupValues()()
			ctx.SetValue("gsuite_sso_domain", domain)

			if strings.HasSuffix(ctx.AuthUser.GetEmail(), "@"+domain) {
				return ctx.AuthUser.GetEmail(), nil
			}
			if db := ctx.DB.Model(&user.UserAuthAlias{}).
				Where("user_id = ? AND id LIKE ?", u.ID, "%@"+domain).
				Order("id ASC").
				PluckFirst("id", &nameID); db.RecordNotFound() {
				err = ctx.ErrorT(ErrUserDoesNotHaveMailForDomain)
			} else {
				err = db.Error
			}
		}
	}
	return
})

func Finder() ect_samlidp.NameIDFinder {
	return finder
}
