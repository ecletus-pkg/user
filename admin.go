package user

import (
	"net/http"
	"sort"
	"strings"

	"github.com/ecletus/admin/admin_helpers"

	"github.com/ecletus/core/resource"

	"github.com/ecletus/admin"
	"github.com/ecletus/core"
	"github.com/ecletus/notification"
	"github.com/ecletus/plug"
	"github.com/ecletus/roles"
)

func (p *Plugin) userSetup(res *admin.Resource, options *plug.Options, Notification *notification.Notification, logouters *Logouters) {
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

	res.IndexAttrs("Email", "Name", "Role", "AdvancedRolesString")
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

	res.Action(&admin.Action{
		Name:   ActionLogout,
		Method: http.MethodPost,
		Type:   admin.ActionDanger,
		Modes:  []string{"menu_item"},
		Handler: func(argument *admin.ActionArgument) error {
			for _, user := range argument.FindSelectedRecords() {
				for _, logouter := range *logouters {
					logouter.Logout(user, argument.Context)
				}
			}
			return nil
		},
	})

	if res.Controller.IsBulkDeleter() {
		res.Action(&admin.Action{
			Name:   ActionBulkLogout,
			Method: http.MethodPost,
			Type:   admin.ActionDanger,
			Handler: func(argument *admin.ActionArgument) error {
				for _, user := range argument.FindSelectedRecords() {
					for _, logouter := range *logouters {
						logouter.Logout(user, argument.Context)
					}
				}
				return nil
			},
			Modes: []string{"index"},
		})
	}
}

func (p *Plugin) passwordSetup(r, res *admin.Resource, Notification *notification.Notification) {
	SetUserAdminMenuEnabled(res)
	res.Meta(&admin.Meta{Name: "YourPassword", Type: "password"})
	res.Meta(&admin.Meta{Name: "NewPassword", Type: "password"})
	res.Meta(&admin.Meta{Name: "PasswordConfirm", Type: "password"})
}

func (p *Plugin) groupSetup(res *admin.Resource, options *plug.Options, Notification *notification.Notification) {
	res.IndexAttrs("Name", "Description")
	res.ShowAttrs("Name", "Description")
	res.NewAttrs("Name", "Description")
	res.EditAttrs(res.ShowAttrs())

	res.AddResourceField("Users", nil, func(res *admin.Resource) {
		admin_helpers.SelectOneOption(admin_helpers.SelectConfigOptionBottonSheet, res, "User")
		res.IndexAttrs("User")
		res.ShowAttrs("User")
		res.NewAttrs("User")
		res.EditAttrs(res.ShowAttrs())
	})

	res.Permission = roles.NewPermission().
		DenyAnother(roles.Create, p.Config.CreateRole).
		DenyAnother(roles.Read, p.Config.ReadRole).
		DenyAnother(roles.Update, p.Config.UpdateRole).
		DenyAnother(roles.Delete, p.Config.DeleteRole)
}

func GetResource(Admin *admin.Admin) *admin.Resource {
	return Admin.GetResourceByID("User")
}

func GetGroupResource(Admin *admin.Admin) *admin.Resource {
	return Admin.GetResourceByID("Group")
}

func SetUserAdminMenuEnabled(res *admin.Resource) {
	menuEnabled := res.DefaultMenu().Enabled
	res.DefaultMenu().Enabled = func(menu *admin.Menu, context *admin.Context) bool {
		if user := context.CurrentUser(); user != nil {
			return context.HasRole(admin.ROLE) && (menuEnabled == nil || menuEnabled(menu, context))
		}
		return false
	}
}
