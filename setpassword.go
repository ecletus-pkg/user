package user

import (
	"errors"

	"github.com/ecletus/auth"

	"github.com/ecletus/auth/auth_identity"
	"github.com/ecletus/auth/providers/password"

	"github.com/ecletus/admin"
	"github.com/ecletus/notification"
)

type SetPasswordController struct {
	UserResource *admin.Resource
	Notification *notification.Notification
}

func (c *SetPasswordController) Update(ctx *admin.Context, recorde interface{}) {
	adminUser := ctx.CurrentUser()
	if adminUser == nil || adminUser.GetName() != "admin" {
		ctx.AddError(errors.New(ctx.Ts(I18n("set_password.unauthorized"))))
		return
	}
	sp := recorde.(*SetPassword)
	if sp.YourPassword == "" {
		ctx.AddError(errors.New(ctx.Ts(I18n("set_password.your_password_is_blank"))))
		return
	}
	if len(sp.NewPassword) < 6 {
		ctx.AddError(errors.New(ctx.Ts(auth.I18n("passwords.password_too_weak"))))
		return
	}
	if sp.NewPassword != sp.PasswordConfirm {
		ctx.AddError(errors.New(ctx.Ts(auth.I18n("passwords.passwords_not_match"))))
		return
	}
	var user User
	if err := c.UserResource.Crud(ctx.Context.CloneBasic()).FindOne(&user, ctx.ParentResourceID[0]); err != nil {
		ctx.AddError(err)
		return
	}

	updater := password.PasswordUpdater{
		UID:               user.GetEmail(),
		UserID:            user.GetID(),
		Provider:          ctx.Admin.Auth.Auth().GetProvider("password").(*password.Provider),
		DB:                ctx.DB,
		NewPassword:       sp.NewPassword,
		PasswordConfirm:   sp.NewPassword,
		CurrentPassword:   sp.YourPassword,
		Confirmed:         true,
		Createable:        true,
		AuthIdentityModel: &auth_identity.AuthIdentity{},
		AdminUserUID:      adminUser.GetEmail(),
		T:                 ctx.Ts,
	}

	if err := updater.Update(); err != nil {
		ctx.AddError(err)
		return
	}

	ctx.RedirectTo = c.UserResource.GetContextURI(ctx.Context, user.ID)
}
