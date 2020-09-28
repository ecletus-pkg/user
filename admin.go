package user

import (
	"sort"

	"github.com/ecletus/admin/admin_helpers"

	"github.com/ecletus/notification"
	"github.com/ecletus/plug"

	"github.com/ecletus/admin"
	"github.com/ecletus/core"
	"github.com/ecletus/roles"
)

func (p *Plugin) userSetup(res *admin.Resource, options *plug.Options, Notification *notification.Notification, logouters *Logouters) {
	res.DefaultMenu().Priority = -2
	rolesSlice := append([]string{}, options.GetStrings(p.RolesKey)...)
	sort.Strings(rolesSlice)

	res.Meta(&admin.Meta{Name: "Confirmed", Valuer: func(user interface{}, ctx *core.Context) interface{} {
		if user.(*User).ID.IsZero() {
			return true
		}
		return user.(*User).Confirmed
	}})

	res.AfterRegister(func() {
		res.GetAdminLayout(admin.BASIC_LAYOUT).SetMetaNames("ID", "Name")
	})

	res.ShowAttrs(
		&admin.Section{
			Title: "Basic Information",
			Rows: [][]string{
				{"Name", "Email"},
				{"Roles"},
				{"Confirmed"},
			},
		},
	)
	res.NewAttrs(res.ShowAttrs(), "-AdvancedRoles")
	res.EditAttrs(res.ShowAttrs(), "-AdvancedRoles")
	res.Permission = roles.NewPermission().Allow(roles.CRUD, roleUserManager)

	/*
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
	*/

	res.AddResourceField("AuthAliases", nil, func(res *admin.Resource) {
		res.Meta(&admin.Meta{Name: "ID", Type: "string"})
		res.INESAttrs("ID", "Description")
	})

	actionSetPassword(res)
}

func (p *Plugin) groupSetup(res *admin.Resource, options *plug.Options, Notification *notification.Notification) {
	res.IndexAttrs("Name", "Description")
	res.ShowAttrs("Name", "Description")
	res.NewAttrs("Name", "Description")
	res.EditAttrs(res.ShowAttrs())
	res.DefaultMenu().Icon = "Group"

	res.AddResourceField("Users", nil, func(res *admin.Resource) {
		admin_helpers.SelectOneOption(admin_helpers.SelectConfigOptionBottonSheet, res, admin_helpers.NameCallback{Name: "User"})
		res.IndexAttrs("User")
		res.ShowAttrs("User")
		res.NewAttrs("User")
		res.EditAttrs(res.ShowAttrs())
	})

	res.Permission = roles.NewPermission().Allow(roles.CRUD, roleUserManager)
}

func GetResource(Admin *admin.Admin) *admin.Resource {
	return Admin.GetResourceByID("User")
}

func GetGroupResource(Admin *admin.Admin) *admin.Resource {
	return Admin.GetResourceByID("Group")
}

func SetUserAdminMenuEnabled(res *admin.Resource) {
	menuEnabled := res.DefaultMenu().Enabled
	res.DefaultMenu().Enabled = func(menu *admin.Menu, context *admin.Context) (ok bool) {
		if _, ok := context.Result.(*User); !ok {
			return false
		}
		if user := context.CurrentUser(); user != nil {
			if !context.IsSuperUser() {
				return
			}
			return menuEnabled == nil || menuEnabled(menu, context)
		}
		return false
	}
}
