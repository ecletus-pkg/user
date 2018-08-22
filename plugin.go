package user

import (
	"fmt"
	"syscall"

	"github.com/aghape/admin"
	"github.com/aghape/admin/adminplugin"
	"github.com/aghape/core"
	"github.com/aghape/auth"
	"github.com/aghape/auth/auth_identity"
	"github.com/aghape/auth/providers/password"
	"github.com/aghape/cli"
	"github.com/aghape/db"
	"github.com/aghape/helpers"
	"github.com/aghape/plug"
	"github.com/aghape/sites"
	"github.com/moisespsena/go-error-wrap"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"
)

var (
	USER_MENU = PKG + ".userMenu"
)

type Plugin struct {
	plug.EventDispatcher
	db.DBNames
	adminplugin.AdminNames
	SitesReaderKey, AuthKey string
}

func (p *Plugin) OnRegister(options *plug.Options) {
	if p.SitesReaderKey == "" {
		panic("SitesReaderKey is BLANK")
	}
	if p.AuthKey == "" {
		panic("AuthKey is BLANK")
	}
	p.AdminNames.OnInitResources(p, func(e *adminplugin.AdminEvent) {
		menu := options.GetStrings(USER_MENU)
		e.Admin.AddResource(&User{}, &admin.Config{Setup: func(res *admin.Resource) {
			p.userSetup(res, options)
		}, Menu: menu})
	})

	db.Events(p).DBOnMigrateGorm(func(e *db.GormDBEvent) error {
		return helpers.CheckReturnE(func() (key string, err error) {
			return "Migrate", e.DB.AutoMigrate(&User{}).Error
		}, func() (key string, err error) {
			return "Create Index", e.DB.Model(&User{}).AddUniqueIndex("idx_user_name", "name").Error
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

			for _, dbName := range p.DBNames.GetNames() {
				log.Infof("Site %q, DB %q: Redefinindo a senha do usuário %q.", site.Name(), dbName, name)

				identity := &auth_identity.AuthIdentity{}
				DB := site.GetDB(dbName)
				DB.DB.Find(identity, "uid = ?", name)

				if identity.ID == 0 {
					log.Infof("Site %q, DB %q: Usuário %q não exists.", site.Name(), dbName, name)
				}

				provider := Auth.GetProvider("password").(*password.Provider)

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

				hashedPassword, _ := provider.Encryptor.Digest(passwd)
				identity.EncryptedPassword = hashedPassword

				if err = DB.DB.Save(identity).Error; err != nil {
					return errwrap.Wrap(err, "Save on DB")
				}

				log.Infof("Site %q, DB %q: Usuário %q done.", site.Name(), dbName, name)
			}
			return nil
		})

		rupCmd.Flags().StringP("password", "p", "", "The new password")
		rupCmd.Flags().BoolP("read-password", "P", false, "Read passsword from STDIN")

		e.RootCmd.AddCommand(rupCmd)
	})
}
