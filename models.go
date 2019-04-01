package user

import (
	"strings"
	"time"

	"github.com/ecletus/fragment"
	"github.com/ecletus/media/oss"
	"github.com/ecletus/roles"
	"github.com/ecletus/validations"
	"github.com/moisespsena-go/aorm"
)

type AdvancedRoles map[string]map[string]bool

func (r AdvancedRoles) Has(group string, name ...string) bool {
	if name == nil {
		parts := strings.Split(group, ":")
		if len(parts) == 1 {
			if items, ok := r[group]; ok && len(items) > 0 {
				return true
			}
			return false
		}
		group = parts[0]
		name = []string{parts[1]}
	}
	if items, ok := r[group]; ok {
		for _, name := range name {
			if _, ok := items[name]; ok {
				return true
			}
		}
	}
	return false
}

func (r *AdvancedRoles) Add(group string, name ...string) {
	if name == nil {
		parts := strings.Split(group, ":")
		group = parts[0]
		name = []string{parts[1]}
	}
	if _, ok := (*r)[group]; !ok {
		(*r)[group] = map[string]bool{}
	}
	for _, name := range name {
		(*r)[group][name] = true
	}
}

func (r AdvancedRoles) Strings() (names []string) {
	for group, items := range r {
		for name := range items {
			names = append(names, group+":"+name)
		}
	}
	return
}

type User struct {
	aorm.AuditedModel
	aorm.VirtualFields

	fragment.FragmentedModel

	Email string `form:"email"`
	Name  string `form:"name"`
	Role  string

	// Confirm
	ConfirmToken string
	Confirmed    bool

	// Recover
	RecoverToken       string
	RecoverTokenExpiry *time.Time

	systemAdmin bool

	DefaultLocale string
	Locales       []string
	AdvancedRoles AdvancedRoles `sql:"-";gorm:"-"`
}

func (user *User) AormAfterInlinePreloadScan(ip *aorm.InlinePreloader, recorde, value interface{}) {
	if _, ok := value.(roles.Roler); ok {
		user.ReloadRoles()
	}
}

func (user *User) ReloadRoles() {
	rols := make(AdvancedRoles)
	for id, f := range user.FormFragments {
		if f.Enabled() {
			if roler, ok := f.(roles.Roler); ok {
				if items := roler.GetRoles(); len(items) > 0 {
					rols.Add(id, items...)
				}
			}
		}
	}
	user.AdvancedRoles = rols
}

func (user *User) GetRoles() (rols []string) {
	if user.Role != "" {
		rols = append(rols, user.Role)
	}

	rols = append(rols, user.AdvancedRoles.Strings()...)
	return
}

func (user *User) GetAdvancedRoles() AdvancedRoles {
	return user.AdvancedRoles
}

func (user *User) SetFormFragment(super fragment.FragmentedModelInterface, id string, value fragment.FormFragmentModelInterface) {
	user.FragmentedModel.SetFormFragment(super, id, value)
	if value == nil {
		user.ReloadRoles()
	} else if _, ok := value.(roles.Roler); ok {
		user.ReloadRoles()
	}
}

func (user *User) GetGormInlinePreloadFields() []string {
	return []string{"Name", "Email"}
}

func (user *User) GetDefaultLocale() string {
	return user.DefaultLocale
}

func (user *User) GetLocales() []string {
	return user.Locales
}

func (user *User) SystemAdmin() {
	user.systemAdmin = true
	user.Name = "admin"
	user.Confirmed = true
	user.Role = "Admin"
	if user.Email == "" {
		user.Email = "admin@localhost.com"
	}
}

func (user *User) String() string {
	return user.Name
}

func (user *User) GetName() string {
	return user.Name
}

func (user *User) DisplayName() string {
	return user.Email
}

func (user *User) GetEmail() string {
	return user.Email
}

func (user *User) AvailableLocales() []string {
	return []string{"pt-BR"}
}

func (user *User) Validate(db *aorm.DB) {
	if !user.systemAdmin && user.Name == "admin" {
		db.AddError(validations.Failed(user, "Name", "Invalid name."))
	}
}

func (user *User) GetVirtualField(name string) (v interface{}, ok bool) {
	if v, ok = user.VirtualFields.GetVirtualField(name); ok {
		return
	}
	return user.FragmentedModel.GetVirtualField(name)
}

type AvatarImageStorage struct{ oss.OSS }

func (AvatarImageStorage) GetSizes() map[string]*oss.Size {
	return map[string]*oss.Size{
		"small":  {Width: 50, Height: 50},
		"middle": {Width: 120, Height: 120},
		"big":    {Width: 320, Height: 320},
	}
}

type SetPassword struct {
	YourPassword    string
	NewPassword     string
	PasswordConfirm string
}

type UserRole struct {
	fragment.FormFragmentModel

	Role string
}

func (ur *UserRole) GetRoles() []string {
	if ur.Role != "" {
		return []string{ur.Role}
	}
	return nil
}

type Group struct {
	aorm.AuditedModel
	aorm.VirtualFields

	fragment.FragmentedModel

	Name, Description string

	Users []UserGroup `gorm:"foreignkey:GroupID"`
}

type UserGroup struct {
	aorm.AuditedModel
	UserID  string `gorm:"size:24;index;unique_index:ux_user_group"`
	User    *User
	GroupID string `gorm:"size:24;index;unique_index:ux_user_group"`
	Group   *Group
}

func (UserGroup) GetGormInlinePreloadFields() []string {
	return []string{"*", "User"}
}
