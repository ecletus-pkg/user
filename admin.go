package user

import (
	"sort"
	"strings"

	"github.com/aghape/core/resource"

	"github.com/aghape/admin"
	"github.com/aghape/core"
	"github.com/aghape/notification"
	"github.com/aghape/plug"
	"github.com/aghape/roles"
)

func (p *Plugin) userSetup(res *admin.Resource, options *plug.Options, Notification *notification.Notification) {
	rolesSlice := append([]string{}, options.GetStrings(p.RolesKey)...)
	sort.Strings(rolesSlice)

	res.Meta(&admin.Meta{Name: "Role", Config: &admin.SelectOneConfig{Collection: rolesSlice, AllowBlank: true}})
	res.Meta(&admin.Meta{Name: "AdvancedRolesString", Valuer: func(recorde interface{}, context *core.Context) interface{} {
		if recorde != nil {
			u := recorde.(*User)
			if u.AdvancedRoles != nil {
				var data []string
				for fragName, roles := range u.AdvancedRoles {
					fres := res.Fragments.Get(fragName).Resource
					var names []string
					for name := range roles {
						names = append(names, context.Ts(fres.I18nKey()+".roles."+name, name))
					}
					sort.Strings(names)
					data = append(data, context.Ts(fres.SingularLabelKey(), fragName)+": "+strings.Join(names, ", "))
				}
				return strings.Join(data, "; ")
			}
		}
		return ""
	}, Setter: func(recorde interface{}, metaValue *resource.MetaValue, context *core.Context) error {
		return nil
	}})

	res.Meta(&admin.Meta{Name: "Confirmed", Valuer: func(user interface{}, ctx *core.Context) interface{} {
		if user.(*User).ID == "" {
			return true
		}
		return user.(*User).Confirmed
	}})

	res.AfterRegister(func() {
		res.GetAdminLayout(admin.BASIC_LAYOUT).SetMetaNames("Name")
	})

	res.Filter(&admin.Filter{
		Name: "Role",
		Config: &admin.SelectOneConfig{
			Collection: rolesSlice,
		},
	})

	res.IndexAttrs("ID", "Email", "Name", "Role", "AdvancedRolesString")
	res.ShowAttrs(
		&admin.Section{
			Title: "Basic Information",
			Rows: [][]string{
				{"Name", "Email"},
				{"Role"},
				{"Confirmed"},
			},
		},
	)
	res.NewAttrs(res.ShowAttrs(), "-AdvancedRoles")
	res.EditAttrs(res.ShowAttrs(), "-AdvancedRoles")
	res.Permission = roles.NewPermission().
		DenyAnother(roles.Create, p.Config.CreateRole).
		DenyAnother(roles.Read, p.Config.ReadRole).
		DenyAnother(roles.Update, p.Config.UpdateRole).
		DenyAnother(roles.Delete, p.Config.DeleteRole)
}

func (p *Plugin) passwordSetup(r, res *admin.Resource, Notification *notification.Notification) {
	menuEnabled := res.DefaultMenu().Enabled
	res.DefaultMenu().Enabled = func(menu *admin.Menu, context *admin.Context) bool {
		if user := context.CurrentUser(); user != nil {
			return user.GetName() == "admin" && menuEnabled(menu, context)
		}
		return false
	}
	res.Meta(&admin.Meta{Name: "YourPassword", Type: "password"})
	res.Meta(&admin.Meta{Name: "NewPassword", Type: "password"})
	res.Meta(&admin.Meta{Name: "PasswordConfirm", Type: "password"})
}

func GetResource(Admin *admin.Admin) *admin.Resource {
	return Admin.GetResourceByID("User")
}
