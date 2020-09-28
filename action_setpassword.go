package user

import (
	"errors"
	"strings"

	"github.com/ecletus/admin"
	"github.com/ecletus/auth"
	"github.com/ecletus/auth/auth_identity"
	"github.com/ecletus/auth/providers/password"
	"github.com/moisespsena-go/aorm"
)

func actionSetPassword(res *admin.Resource) {
	actionRes := res.Admin.NewResource(&SetPassword{})
	actionRes.Permission = res.Permission
	res.Action(&admin.Action{
		Name:     "Set Password",
		Modes:    []string{"menu_item", "show"},
		Resource: actionRes,
		Available: func(context *admin.Context) bool {
			return true
		},
		SetupArgument: func(arg *admin.ActionArgument) (err error) {
			var (
				ctx        = arg.Context
				sp         = arg.Argument.(*SetPassword)
				targetUser User
			)
			if err = res.Crud(ctx.Context.CloneBasic()).FindOne(&targetUser, ctx.ResourceID); err != nil {
				return err
			}

			if targetUser.IsSuper() && !arg.Context.CurrentUser().IsSuper() {
				return auth.ErrUnauthorized
			}

			sp.targetUser = &targetUser
			return
		},
		Handler: func(arg *admin.ActionArgument) (err error) {
			var (
				ctx       = arg.Context
				adminUser = ctx.CurrentUser()
				sp        = arg.Argument.(*SetPassword)
			)

			if sp.YourPassword == "" {
				return errors.New(ctx.Ts(I18n("set_password.your_password_is_blank")))
			}
			if len(sp.NewPassword) < 6 {
				return errors.New(ctx.Ts(auth.I18n("passwords.password_too_weak")))
			}
			if sp.NewPassword != sp.PasswordConfirm {
				return errors.New(ctx.Ts(auth.I18n("passwords.passwords_not_match")))
			}
			updater := password.PasswordUpdater{
				UID:             sp.targetUser.GetUID(),
				UserID:          aorm.IdOf(sp.targetUser).String(),
				Provider:        ctx.Admin.Auth.Auth().GetProvider("password").(*password.Provider),
				DB:              ctx.DB(),
				NewPassword:     sp.NewPassword,
				PasswordConfirm: sp.NewPassword,
				// TODO: Configurable StrengthLevel
				StrengthLevel:     2,
				CurrentPassword:   sp.YourPassword,
				Confirmed:         true,
				Createable:        true,
				AuthIdentityModel: &auth_identity.AuthIdentity{},
				AdminUserUID:      adminUser.GetUID(),
				T:                 ctx.Ts,
			}

			_, authCtx := ctx.Admin.Auth.Auth().NewContextFromRequest(ctx.Request)
			err = updater.Update(authCtx)
			return
		},
	})
}

type SetPassword struct {
	YourPassword    string `admin:"required;type:password"`
	NewPassword     string `admin:"required;type:password"`
	PasswordConfirm string `admin:"required;type:password"`
	targetUser      *User
}

func (SetPassword) AdminResourceSetup(res *admin.Resource, defaultSetup func()) {
	defaultSetup()
	res.Meta(&admin.Meta{
		Name: "YourPassword",
		RecordLabelFormatter: func(meta *admin.Meta, ctx *admin.Context, record interface{}, s string) string {
			user := ctx.CurrentUser()
			return strings.ReplaceAll(s, "%s", user.GetName()+" - "+user.DisplayName())
		},
	})

	var newPassFmt = func(meta *admin.Meta, ctx *admin.Context, record interface{}, s string) string {
		user := ctx.Result.(*admin.ActionArgument).Argument.(*SetPassword).targetUser
		return strings.ReplaceAll(s, "%s", user.GetName()+" - "+user.DisplayName())
	}

	res.Meta(&admin.Meta{
		Name:                 "NewPassword",
		RecordLabelFormatter: newPassFmt,
	})
	res.Meta(&admin.Meta{
		Name:                 "PasswordConfirm",
		RecordLabelFormatter: newPassFmt,
	})
}
