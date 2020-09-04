package user

import (
	"github.com/ecletus/auth"
	"github.com/ecletus/auth/auth_identity"
	"github.com/ecletus/auth/providers/password"
	"github.com/ecletus/core"
	"github.com/ecletus/media/oss"
	"github.com/ecletus/notification"
	"github.com/moisespsena-go/aorm"
	defaultlogger "github.com/moisespsena-go/default-logger"
	errwrap "github.com/moisespsena-go/error-wrap"
	path_helpers "github.com/moisespsena-go/path-helpers"
)

var log = defaultlogger.GetOrCreateLogger(path_helpers.GetCalledDir())

const DEFAULT_PASSWORD = "123@456"

func SetUserPassword(site *core.Site, DB *aorm.DB, Auth *auth.Auth, Notification *notification.Notification,
	user auth.User, passwd string, t ...func(key string, defaul ...interface{}) string) (err error) {
	var T func(key string, defaul ...interface{}) string
	if len(t) > 0 && t[0] != nil {
		T = t[0]
	} else {
		T = func(key string, defaul ...interface{}) string {
			return key
		}
	}
	updater := password.PasswordUpdater{
		UID:                     user.Schema().UID,
		UserID:                  aorm.IdOf(user).String(),
		Provider:                Auth.GetProvider("password").(*password.Provider),
		DB:                      DB,
		NewPassword:             passwd,
		PasswordConfirm:         passwd,
		CurrentPasswordDisabled: true,
		Confirmed:               true,
		AuthIdentityModel:       &auth_identity.AuthIdentity{},
		Createable:              true,
		StrengthDisabled:        true,
		T:                       T,
	}

	if err = updater.Update(Auth.NewContextFromSite(site)); err != nil {
		return
	}

	if Notification != nil {
		// Send notification
		_ = Notification.Send(&notification.Message{
			From:        user,
			To:          user,
			Title:       T(auth.I18N_GROUP+".password_changed", "Password changed!"),
			Body:        T(auth.I18N_GROUP+".you_password_changed", "You Password has be changed!"),
			MessageType: "info",
		}, (&core.Context{}).SetDB(DB))
	}
	return nil
}

func CreateAdminUserIfNotExists(site *core.Site, Auth *auth.Auth, Notification *notification.Notification,
	readEmail func() (string, error), readPassword func() (string, error)) (err error) {
	T := func(key string, defaul ...interface{}) string {
		if len(defaul) > 0 {
			if s, ok := defaul[0].(string); ok && s != "" {
				return s
			}
		}
		return key
	}
	var adminUser User
	DB := oss.IgnoreCallback(site.GetSystemDB().DB)
	if err = DB.First(&adminUser, "name = ?", AdminUserName).Error; aorm.IsRecordNotFoundError(err) {
		log.Info("Create System Administrator user")
		var email string

		if email, err = readEmail(); err != nil {
			return errwrap.Wrap(err, "Read admin user mail")
		}

		AdminUser := &User{Email: email}
		AdminUser.SystemAdmin()

		if err = DB.Create(AdminUser).Error; err != nil {
			return errwrap.Wrap(err, "Create admin user into DB")
		}

		var passwd string
		if readPassword != nil {
			if passwd, err = readPassword(); err != nil {
				return errwrap.Wrap(err, "Read admin user password")
			}
		}

		if passwd == "" {
			passwd = DEFAULT_PASSWORD
		}

		updater := password.PasswordUpdater{
			UID:                     AdminUser.GetUID(),
			UserID:                  aorm.IdOf(AdminUser).String(),
			Provider:                Auth.GetProvider("password").(*password.Provider),
			DB:                      DB,
			NewPassword:             passwd,
			PasswordConfirm:         passwd,
			CurrentPasswordDisabled: true,
			Confirmed:               true,
			Createable:              true,
			StrengthDisabled:        true,
			IsAdmin:                 true,
			AuthIdentityModel:       &auth_identity.AuthIdentity{},
			T:                       T,
		}

		if err = updater.Update(Auth.NewContextFromSite(site)); err != nil {
			return err
		}

		if Notification != nil {
			// Send notification
			_ = Notification.Send(&notification.Message{
				From:        AdminUser,
				To:          AdminUser,
				Title:       T(auth.I18N_GROUP+".password_changed", "Password changed!"),
				Body:        T(auth.I18N_GROUP+".you_password_changed", "You Password has be changed!"),
				MessageType: "info",
			}, (&core.Context{}).SetDB(DB))

			log.Infof("Admin User: name=admin, email=%q, password=%q", email, passwd)

			// Send welcome notification
			return Notification.Send(&notification.Message{
				From:        AdminUser,
				To:          AdminUser,
				Title:       "Welcome!",
				Body:        "Welcome!",
				MessageType: "info",
			}, (&core.Context{}).SetDB(DB))
		}
	}
	return
}
