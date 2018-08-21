package user

import (
	"git.moisespsena.com/moisespsena/sam/app/models"
	"github.com/aghape/admin"
	"github.com/aghape/aghape"
	"github.com/aghape/aghape/resource"
	"github.com/aghape/plug"
	"github.com/aghape/validations"
	"golang.org/x/crypto/bcrypt"
)

var (
	OPT_ROLES = PKG + ".roles"
)

func (p *Plugin) userSetup(res *admin.Resource, options *plug.Options) {
	roles := append([]string{"Admin"}, options.GetStrings(OPT_ROLES)...)
	res.Meta(&admin.Meta{Name: "Role", Config: &admin.SelectOneConfig{Collection: roles}})
	res.Meta(&admin.Meta{
		Name:            "Password",
		Type:            "password",
		FormattedValuer: func(interface{}, *qor.Context) interface{} { return "" },
		Setter: func(resource interface{}, metaValue *resource.MetaValue, context *qor.Context) error {
			values := metaValue.Value.([]string)
			if len(values) > 0 {
				if newPassword := values[0]; newPassword != "" {
					bcryptPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
					if err != nil {
						context.DB.AddError(validations.Failed(res, "Password", "Can't encrpt password"))
						return nil
					}
					u := resource.(*models.User)
					u.Password = string(bcryptPassword)
				}
			}
			return nil
		},
	})
	res.Meta(&admin.Meta{Name: "Confirmed", Valuer: func(user interface{}, ctx *qor.Context) interface{} {
		if user.(*models.User).ID == 0 {
			return true
		}
		return user.(*models.User).Confirmed
	}})

	res.Filter(&admin.Filter{
		Name: "Role",
		Config: &admin.SelectOneConfig{
			Collection: roles,
		},
	})

	res.IndexAttrs("ID", "Email", "Name", "Role")
	res.ShowAttrs(
		&admin.Section{
			Title: "Basic Information",
			Rows: [][]string{
				{"Name"},
				{"Email", "Password"},
				{"Avatar"},
				{"Role"},
				{"Confirmed"},
			},
		},
		&admin.Section{
			Title: "Accepts",
			Rows: [][]string{
				{"AcceptPrivate", "AcceptLicense", "AcceptNews"},
			},
		},
	)
	res.EditAttrs(res.ShowAttrs())
}

func GetResource(Admin *admin.Admin) *admin.Resource {
	return Admin.GetResourceByID("User")
}