package user

import (
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/ecletus/auth/auth_identity/helpers"

	"github.com/ecletus/auth"
	"github.com/ecletus/auth/auth_identity"
	"github.com/ecletus/auth/providers/password"
	"github.com/ecletus/core"
	"github.com/ecletus/sites"
	"github.com/moisespsena-go/aorm"
	errwrap "github.com/moisespsena-go/error-wrap"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"
)

func CreateCommands(register *core.SitesRegister, authGetter func() *auth.Auth, preRun ...func()) (cmds []*cobra.Command, err error) {
	cu := &sites.CmdUtils{SitesRegister: register}
	var userCmd = &cobra.Command{
		Use:   "user",
		Short: "User management",
	}
	var (
		pwdReseter *cobra.Command
		adminFlag  = AdminFlagCommand(cu, authGetter, preRun...)
	)
	if pwdReseter, err = PasswordResetterCommand(cu, authGetter, preRun...); err != nil {
		return
	}

	adminFlag.AddCommand(
		AdminFlagCommandChanger(cu, authGetter, true, preRun...),
		AdminFlagCommandChanger(cu, authGetter, false, preRun...),
	)

	userCmd.AddCommand(
		pwdReseter,
		adminFlag,
	)

	return []*cobra.Command{userCmd}, nil
}

func PasswordResetterCommand(cu *sites.CmdUtils, authGetter func() *auth.Auth, preRun ...func()) (cmd *cobra.Command, err error) {
	cmd = cu.Site(&cobra.Command{
		Use:   "password-reset NAME|EMAIL",
		Short: "Reset the user password",
		Args:  cobra.ExactArgs(1),
	}, func(cmd *cobra.Command, site *core.Site, args []string) (err error) {
		name := args[0]
		if name == "" {
			return fmt.Errorf("User name is empty.")
		}
		for _, f := range preRun {
			f()
		}
		Auth := authGetter()
		DB := site.GetSystemDB().DB
		log.Infof("Site %q: Redefinindo a senha do usuário %q.", site.Name(), name)

		provider := Auth.GetProvider("password").(*password.Provider)

		var user User
		var query string
		if strings.ContainsRune(name, '@') {
			query = "email"
		} else {
			query = "name"
		}

		if err = DB.First(&user, query+" = ?", name).Error; err != nil {
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
			UID:                     user.GetUID(),
			UserID:                  aorm.IdOf(user).String(),
			Provider:                provider,
			DB:                      DB,
			NewPassword:             passwd,
			PasswordConfirm:         passwd,
			CurrentPasswordDisabled: true,
			Confirmed:               true,
			Createable:              true,
			StrengthDisabled:        true,
			AuthIdentityModel:       &auth_identity.AuthIdentity{},
			T: func(key string, defaul ...interface{}) string {
				return key
			},
			NotifyMailer: provider.UpdatedPasswordNotifier,
		}

		if err = updater.Update(Auth.NewContextFromSite(site)); err != nil {
			return err
		}
		return nil
	})

	cmd.Flags().StringP("password", "p", "", "The new password")
	cmd.Flags().BoolP("read-password", "P", false, "Read password from STDIN")
	return cmd, nil
}

func AdminFlagCommand(cu *sites.CmdUtils, authGetter func() *auth.Auth, preRun ...func()) (cmd *cobra.Command) {
	return cu.Site(&cobra.Command{
		Use:   "admin-flag NAME|EMAIL",
		Short: "show admin flag",
		Args:  cobra.ExactArgs(1),
	}, func(cmd *cobra.Command, site *core.Site, args []string) (err error) {
		var (
			user     *User
			Auth     *auth.Auth
			identity auth_identity.AuthIdentityInterface
		)

		for _, f := range preRun {
			f()
		}

		if user, Auth, err = GetUserAndAuth(fmt.Sprintf("%v admin flag for user %q.", cmd.Name(), args[0]), site, args[0], authGetter); err != nil {
			return
		}

		if identity, err = helpers.GetIdentity(
			Auth.AuthIdentityModel,
			password.Name,
			site.GetSystemDB().DB, user.GetUID(),
		); err != nil {
			return
		}
		if identity.GetAuthBasic().IsAdmin {
			fmt.Fprintln(os.Stdout, "admin flag is ENABLED")
		} else {
			fmt.Fprintln(os.Stdout, "admin flag is DISABLED")
		}
		return nil
	})
}

func AdminFlagCommandChanger(cu *sites.CmdUtils, authGetter func() *auth.Auth, enable bool, preRun ...func()) (cmd *cobra.Command) {
	var use string
	if enable {
		use = "enable"
	} else {
		use = "disable"
	}

	return cu.Sites(&cobra.Command{
		Use:   use + " NAME|EMAIL",
		Short: use + " admin flag",
		Args:  cobra.ExactArgs(1),
	}, func(cmd *cobra.Command, site *core.Site, args []string) (err error) {
		var (
			user *User
			Auth *auth.Auth
		)
		for _, f := range preRun {
			f()
		}
		if user, Auth, err = GetUserAndAuth(fmt.Sprintf("%v for user %q.", cmd.Name(), args[0]), site, args[0], authGetter); err != nil {
			return
		}

		return helpers.SetAdminFlag(
			Auth.AuthIdentityModel,
			Auth.GetProvider(password.Name).(*password.Provider),
			site.GetSystemDB().DB,
			user.GetUID(),
			enable,
		)
	})
}

func GetUserAndAuth(logMsg string, site *core.Site, nameOrEmail string, authGetter func() *auth.Auth) (user *User, Auth *auth.Auth, err error) {
	if nameOrEmail == "" {
		return nil, nil, fmt.Errorf("user name or email is empty.")
	}

	Auth = authGetter()
	DB := site.GetSystemDB().DB
	log.Infof("site %q: %s", site.Name(), logMsg)

	if user, err = GetUser(DB, nameOrEmail); err != nil {
		return
	}
	return
}

func GetUser(DB *aorm.DB, nameOrEmail string) (user *User, err error) {
	user = &User{}
	var query string
	if strings.ContainsRune(nameOrEmail, '@') {
		query = "email"
	} else {
		query = "name"
	}
	if err = DB.First(&user, query+" = ?", nameOrEmail).Error; err != nil {
		if aorm.IsRecordNotFoundError(err) {
			return nil, fmt.Errorf("User does not exists.")
		}
		return nil, errwrap.Wrap(err, "Find user")
	}
	return
}
