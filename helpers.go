package user

import (
	"fmt"
	"time"

	"git.moisespsena.com/moisespsena/sam/app/models"
	"github.com/aghape/aghape"
	"github.com/aghape/auth"
	"github.com/aghape/auth/auth_identity"
	"github.com/aghape/auth/providers/password"
	"github.com/aghape/media"
	"github.com/aghape/notification"
	"github.com/moisespsena/go-default-logger"
	"github.com/moisespsena/go-error-wrap"
	"github.com/moisespsena/go-path-helpers"
)

var log = defaultlogger.NewLogger(path_helpers.GetCalledDir())

const DEFAULT_PASSWORD = "123456"

func CreateAdminUserIfNotExists(site qor.SiteInterface, Auth *auth.Auth, Notification *notification.Notification,
	readEmail func() (string, error)) (err error) {
	var adminUser User
	DB := media.IgnoreCallback(site.GetSystemDB().DB)
	DB.First(&adminUser, "name = ?", "admin")
	if adminUser.ID == 0 {
		log.Info("Create System Administrator user")
		email, err := readEmail()
		if err != nil {
			return errwrap.Wrap(err, "Read admin user mail")
		}
		AdminUser := &models.User{Email: email}
		AdminUser.SystemAdmin()
		err = DB.Create(AdminUser).Error
		if err != nil {
			return errwrap.Wrap(err, "Create admin user into DB")
		}

		provider := Auth.GetProvider("password").(*password.Provider)
		hashedPassword, _ := provider.Encryptor.Digest(DEFAULT_PASSWORD)
		now := time.Now()

		authIdentity := &auth_identity.AuthIdentity{}
		authIdentity.Provider = "password"
		authIdentity.UID = AdminUser.Email
		authIdentity.EncryptedPassword = hashedPassword
		authIdentity.UserID = fmt.Sprint(AdminUser.ID)
		authIdentity.ConfirmedAt = &now

		err = DB.Create(authIdentity).Error
		if err != nil {
			return errwrap.Wrap(err, "Create first user")
		}

		log.Info("Admin User: name=admin, email=" + email + ", password=" + DEFAULT_PASSWORD)

		// Send welcome notification
		return Notification.Send(&notification.Message{
			From:        AdminUser,
			To:          AdminUser,
			Title:       "Welcome!",
			Body:        "Welcome!",
			MessageType: "info",
		}, &qor.Context{DB: DB})
	}
	return nil
}
