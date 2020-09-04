package user

import (
	"errors"

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
			var user User
			if err = res.Crud(ctx.Context.CloneBasic()).FindOne(&user, ctx.ResourceID); err != nil {
				return err
			}

			updater := password.PasswordUpdater{
				UID:             user.GetUID(),
				UserID:          aorm.IdOf(user).String(),
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
}
