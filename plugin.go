package user

import (
	"fmt"
	"syscall"

	"github.com/moisespsena-go/aorm"

	"github.com/aghape-pkg/admin"
	"github.com/aghape/admin"
	"github.com/aghape/auth"
	"github.com/aghape/auth/auth_identity"
	"github.com/aghape/auth/providers/password"
	"github.com/aghape/cli"
	"github.com/aghape/core"
	"github.com/aghape/db"
	"github.com/aghape/helpers"
	"github.com/aghape/notification"
	"github.com/aghape/plug"
	"github.com/aghape/sites"
	"github.com/moisespsena/go-error-wrap"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"
)

var (
	USER_MENU = PKG + ".userMenu"
)

type Config struct {
	CreateRole string
	DeleteRole string
	UpdateRole string
	ReadRole   string
}

type Plugin struct {
	plug.EventDispatcher
	db.DBNames
	admin_plugin.AdminNames
	SitesReaderKey, NotificationKey, AuthKey, RolesKey string
	Config                                             Config
}

func (p *Plugin) OnRegister(options *plug.Options) {
	if p.SitesReaderKey == "" {
		panic("SitesReaderKey is BLANK")
	}
	if p.AuthKey == "" {
		panic("AuthKey is BLANK")
	}

	if p.Config.CreateRole == "" {
		p.Config.CreateRole = admin.ROLE
	}

	if p.Config.ReadRole == "" {
		p.Config.ReadRole = admin.ROLE
	}

	if p.Config.UpdateRole == "" {
		p.Config.UpdateRole = admin.ROLE
	}

	if p.Config.DeleteRole == "" {
		p.Config.DeleteRole = admin.ROLE
	}

	admin_plugin.Events(p).InitResources(func(e *admin_plugin.AdminEvent) {
		menu := options.GetStrings(USER_MENU)
		n := options.GetInterface(p.NotificationKey).(*notification.Notification)
		res := e.Admin.AddResource(&User{}, &admin.Config{Setup: func(res *admin.Resource) {
			p.userSetup(res, options, n)
		}, Menu: menu})

		res.AddResource(&admin.SubConfig{}, &SetPassword{}, &admin.Config{
			Singleton:  true,
			Controller: &SetPasswordController{res, n},
			Setup: func(pres *admin.Resource) {
				p.passwordSetup(res, pres, n)
			},
		})

	})

	db.Events(p).DBOnMigrate(func(e *db.DBEvent) error {
		return helpers.CheckReturnE(func() (key string, err error) {
			return "Migrate", e.AutoMigrate(&User{}).Error
		}, func() (key string, err error) {
			return "Create Index", e.Model(&User{}).AddUniqueIndex("idx_user_name", "name").Error
		})
	})

	cli.OnRegister(p, func(e *cli.RegisterEvent) {
		SitesReader := e.Options().GetInterface(p.SitesReaderKey).(core.SitesReaderInterface)
		cmd := &sites.CmdUtils{SitesReader: SitesReader}
		var rupCmd = cmd.Sites(&cobra.Command{
			Use:   "user-password-reset NAME",
			Short: "Reset the user password",
			Args:  cobra.ExactArgs(1),
		}, func(cmd *cobra.Command, site core.SiteInterface, args []string) (err error) {
			name := args[0]
			if name == "" {
				return fmt.Errorf("User name is empty.")
			}

			Auth := options.GetInterface(p.AuthKey).(*auth.Auth)
			DB := site.GetSystemDB().DB
			log.Infof("Site %q: Redefinindo a senha do usuário %q.", site.Name(), name)

			provider := Auth.GetProvider("password").(*password.Provider)

			var user User
			if err = DB.First(&user, "email = ?", name).Error; err != nil {
				if aorm.IsRecordNotFoundError(err) {
					return fmt.Errorf("User does not exists.")
				}
				return errwrap.Wrap(err, "Find user")
			}

			var (
				passwd     string
				readPasswd bool
			)

			if readPasswd, err = cmd.Flags().GetBool("read-password"); err != nil {
				return fmt.Errorf("Read-Password flag failed.")
			}
			if readPasswd {
				fmt.Print("Your password: ")
				bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
				if err != nil {
					return errwrap.Wrap(err, "Read Password")
				}
				passwd = string(bytePassword)
				fmt.Println() // it's necessary to add a new line after user's input
			} else {
				if passwd, err = cmd.Flags().GetString("password"); err != nil {
					return fmt.Errorf("Password flag failed.")
				}
			}

			if passwd == "" {
				return fmt.Errorf("Password is blank.")
			}

			updater := password.PasswordUpdater{
				UID:                     user.GetEmail(),
				UserID:                  user.GetID(),
				Provider:                provider,
				DB:                      DB,
				NewPassword:             passwd,
				PasswordConfirm:         passwd,
				CurrentPasswordDisabled: true,
				Confirmed:               true,
				Createable:              true,
				AuthIdentityModel:       &auth_identity.AuthIdentity{},
				T: func(key string, defaul ...interface{}) string {
					return key
				},
			}

			if err = updater.Update(); err != nil {
				return err
			}
			return nil
		})

		rupCmd.Flags().StringP("password", "p", "", "The new password")
		rupCmd.Flags().BoolP("read-password", "P", false, "Read passsword from STDIN")

		e.RootCmd.AddCommand(rupCmd)
	})
}
