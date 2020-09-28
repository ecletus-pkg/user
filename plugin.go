package user

import (
	admin_plugin "github.com/ecletus-pkg/admin"
	"github.com/ecletus/cli"
	"github.com/ecletus/db"
	"github.com/ecletus/notification"
	"github.com/ecletus/plug"
	"github.com/pkg/errors"

	"github.com/ecletus/admin"
	"github.com/ecletus/auth"
	"github.com/ecletus/auth/auth_identity/helpers"
	"github.com/ecletus/core"
	"github.com/moisespsena-go/aorm"
)

var (
	USER_MENU     = PKG + ".menu.user"
	GROUP_MENU    = PKG + ".menu.group"
	LOGOUTERS_KEY = PKG + ".logouters"
)

type Config struct {
	GroupsDisabled       bool
	AccessTokensDisabled bool
}

type Plugin struct {
	plug.EventDispatcher
	db.DBNames
	admin_plugin.AdminNames

	SitesRegisterKey,
	NotificationKey,
	AuthKey,
	RolesKey,
	LogoutersKey string

	Config Config
}

func (p *Plugin) RequireOptions() []string {
	if p.NotificationKey != "" {
		return []string{p.NotificationKey}
	}
	return nil
}

func (p *Plugin) OnRegister(options *plug.Options) {
	if p.SitesRegisterKey == "" {
		panic("SitesReaderKey is BLANK")
	}
	if p.AuthKey == "" {
		panic("AuthKey is BLANK")
	}

	if p.LogoutersKey == "" {
		p.LogoutersKey = LOGOUTERS_KEY
	}

	var logouters Logouters

	options.Set(p.LogoutersKey, &logouters)

	admin_plugin.Events(p).InitResources(func(e *admin_plugin.AdminEvent) {
		menu := options.GetStrings(USER_MENU)
		var Notification *notification.Notification
		if p.NotificationKey != "" {
			if ni := options.GetInterface(p.NotificationKey); ni != nil {
				Notification = ni.(*notification.Notification)
			}
		}

		e.Admin.OnPreInitializeMeta(func(meta *admin.Meta) {
			if meta.Name == aorm.SoftDeleteFieldDeletedByID {
				// TODO: implement it
			}
		})

		res := e.Admin.AddResource(&User{}, &admin.Config{
			Setup: func(res *admin.Resource) {
				p.userSetup(res, options, Notification, &logouters)
			},
			Menu: menu,
		})

		Auth := options.GetInterface(p.AuthKey).(*auth.Auth)

		if p.Config.AccessTokensDisabled {
			res.Meta(&admin.Meta{
				Name: "AccessTokens",
				Enabled: func(recorde interface{}, context *admin.Context, meta *admin.Meta) bool {
					return false
				},
			})
		} else {
			res.AddResource(&admin.SubConfig{FieldName: "AccessTokens"}, nil, &admin.Config{
				Setup: func(res *admin.Resource) {
					res.NewAttrs("Name", "Description", "Enabled", "ExpireAt", "LimitAccess")
					res.EditAttrs("Name", "Description", "Enabled", "ExpireAt", "LimitAccess")
					res.ShowAttrs(res.EditAttrs(), "Token")
					res.IndexAttrs(res.EditAttrs())
					res.Meta(&admin.Meta{
						Name: "Token",
						Type: "text",
						Config: &admin.TextConfig{
							WordBreak: admin.WordBreakAll,
							Copy:      true,
						},
						ReadOnly: true,
					})

					res.OnAfterDelete(func(ctx *core.Context, recorde interface{}) error {
						uat := recorde.(*UserAccessToken)
						if identity, err := helpers.GetIdentity(Auth.AuthIdentityModel, "user:access_tokens", ctx.DB().New(), uat.ID.String()); err == nil {
							if err = helpers.DeleteIdentity(ctx.DB(), identity); err != nil {
								return errors.Wrap(err, "Delete indentity")
							}
						}
						return nil
					})

					var createOrUpdate = func(ctx *core.Context, recorde interface{}) error {
						uat := recorde.(*UserAccessToken)
						if uat.ID.IsZero() {
							uat.ID.Generate()
						}
						if identity, err := helpers.GetIdentity(Auth.AuthIdentityModel, "user:access_tokens", ctx.DB().New(), uat.ID.String()); err == nil {
							if uat.Enabled {
								basic := identity.GetAuthBasic()
								basic.UID = uat.ID.String()
								basic.ExpireAt = uat.ExpireAt
								basic.LimitAccess = uat.LimitAccess
								identity.SetAuthBasic(*basic)
								Claims := identity.GetAuthBasic().ToClaims()
								if err = helpers.SaveIdentity(ctx.DB(), identity); err != nil {
									return err
								}
								if token, err := Auth.SessionStorer.SignedToken(Claims); err != nil {
									return err
								} else {
									recorde.(*UserAccessToken).Token = token
								}
							} else {
								if err = helpers.DeleteIdentity(ctx.DB(), identity); err != nil {
									return errors.Wrap(err, "Delete indentity")
								}
							}
						} else if err == auth.ErrInvalidAccount {
							if uat.Enabled {
								identity = helpers.NewIdentity(Auth.AuthIdentityModel, "user:access_tokens")
								basic := identity.GetAuthBasic()
								basic.UID = uat.ID.String()
								basic.ExpireAt = uat.ExpireAt
								basic.UserID = uat.UserID.String()
								basic.LimitAccess = uat.LimitAccess
								identity.SetAuthBasic(*basic)
								Claims := identity.GetAuthBasic().ToClaims()
								if err = helpers.SaveIdentity(ctx.DB(), identity); err != nil {
									return err
								}
								if token, err := Auth.SessionStorer.SignedToken(Claims); err != nil {
									return err
								} else {
									recorde.(*UserAccessToken).Token = token
								}
							}
						} else {
							return err
						}
						return nil
					}

					res.OnBeforeCreate(createOrUpdate)
					res.OnBeforeUpdate(func(ctx *core.Context, _, recorde interface{}) error {
						return createOrUpdate(ctx, recorde)
					})
				},
			})
		}

		if !p.Config.GroupsDisabled {
			gmenu := options.GetStrings(GROUP_MENU)
			e.Admin.AddResource(&Group{}, &admin.Config{Setup: func(res *admin.Resource) {
				p.groupSetup(res, options, Notification)
			}, Menu: gmenu})
		}
	})

	db.Events(p).DBOnMigrate(func(e *db.DBEvent) error {
		var values = []interface{}{&User{}, &UserAuthAlias{}, &UserAccessToken{}}
		if !p.Config.GroupsDisabled {
			values = append(values, &Group{}, &UserGroup{})
		}
		return e.AutoMigrate(values...).Error
	})
}

type CliPlugin struct {
	plug.EventDispatcher
	SitesRegisterKey,
	AuthKey,
	AdminGetterKey string
	PreRun []func()
}

func (p *CliPlugin) RequireOptions() []string {
	return []string{p.AdminGetterKey}
}

func (p *CliPlugin) OnRegister(options *plug.Options) {
	cli.OnRegisterE(p, func(e *cli.RegisterEvent) error {
		sitesRegister := e.Options().GetInterface(p.SitesRegisterKey).(*core.SitesRegister)
		cmds, err := CreateCommands(sitesRegister, func() *auth.Auth {
			return options.GetInterface(p.AuthKey).(*auth.Auth)
		}, p.PreRun...)
		if err != nil {
			return err
		}
		e.RootCmd.AddCommand(cmds...)
		return nil
	})
}
