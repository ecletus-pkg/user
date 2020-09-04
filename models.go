package user

import (
	"strings"
	"time"

	"github.com/moisespsena-go/bid"

	"github.com/moisespsena-go/getters"

	"github.com/ecletus/auth"
	"github.com/ecletus/auth/providers/password"
	"github.com/ecletus/roles"

	"github.com/ecletus/fragment"
	"github.com/ecletus/media/oss"
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

type UserAuthAlias struct {
	aorm.DefinedID
	UserID      bid.BID `sql:"index"`
	User        *User
	Description string
}

type UserAccessToken struct {
	aorm.AuditedModel

	UserID            bid.BID `sql:"index"`
	User              *User
	Name, Description string
	ExpireAt          *time.Time
	Enabled           bool
	Token             string
	LimitAccess       uint64
}

type User struct {
	aorm.AuditedSDModel
	fragment.FragmentedModel

	Email string         `form:"email" aorm:"unique;size:64" admin:"required"`
	Name  string         `form:"name" aorm:"unique;size:64" admin:"required"`
	Roles RolesAttribute `aorm:"type:text;default" admin:"select_many;type:select_many"`

	// Confirm
	ConfirmToken string
	Confirmed    bool `aorm:"default"`

	// Recover
	RecoverToken       string `aorm:"size:2048"`
	RecoverTokenExpiry *time.Time

	Super bool `aorm:"default"`

	Location      string        `aorm:"default;size:64"`
	Locale        string        `aorm:"default;size:5"`
	AdvancedRoles AdvancedRoles `sql:"-"`

	AuthAliases  []UserAuthAlias   `aorm:"foreignkey:UserID"`
	AccessTokens []UserAccessToken `aorm:"foreignkey:UserID"`

	_ interface{} `admin:"index_attrs:{Email;Name;Roles;AdvancedRolesString}"`
}

func (this *User) HasRole(name string) (ok bool) {
	if this.Roles.m != nil {
		_, ok = this.Roles.m[name]
	}
	return
}

func (this *User) GetUID() string {
	return this.ID.String()
}

func (this *User) IsSuper() bool {
	return this.Super
}

func (this *User) BeforeAuth(ctx *auth.LoginContext) error {
	if ctx.Provider != nil && ctx.Provider.GetName() == "password" {
		if login, ok := getters.String(ctx.LoginData, password.FieldLogin); ok && login != "" {
			if !strings.ContainsRune(login, '@') {
				// find by name
				if err := ctx.DB().Model(&User{}).
					Where("name = ?", login).
					PluckFirst("email", &login).Error; err == nil {
					ctx.LoginData.Set(password.FieldLogin, login)
					return nil
				} else if !aorm.IsRecordNotFoundError(err) {
					return err
				}
			}
			// find by alias
			if err := ctx.DB().Model(&User{}).
				Joins("JOIN user_auth_aliases ON users.id = user_auth_aliases.user_id").
				Where("user_auth_aliases.id = ?", login).
				PluckFirst("email", &login).Error; err == nil {
				ctx.LoginData.Set(password.FieldLogin, login)
			} else if !aorm.IsRecordNotFoundError(err) {
				return err
			}
		}
	}
	return nil
}

func (*User) FindUID(ctx *auth.Context, identifier string) (uid string, err error) {
	DB := ctx.DB().Model(&User{})
	var userID bid.BID
	// find by email
	if strings.ContainsRune(identifier, '@') {
		if err = DB.
			Where("email = ?", identifier).
			PluckFirst("id", &userID).
			Error; err == nil {
			return userID.String(), nil
		} else if !aorm.IsRecordNotFoundError(err) {
			return
		}
	} else {
		// find by name
		if err = DB.
			Where("name = ?", identifier).
			PluckFirst("id", &userID).
			Error; err == nil {
			return userID.String(), nil
		} else if !aorm.IsRecordNotFoundError(err) {
			return
		}
	}
	// find by alias
	if err = DB.
		Joins("JOIN user_auth_aliases ON users.id = user_auth_aliases.user_id").
		Where("user_auth_aliases.id = ?", identifier).
		PluckFirst("user_id", &userID).
		Error; err == nil {
		return userID.String(), nil
	} else if !aorm.IsRecordNotFoundError(err) {
		return
	}
	return "", auth.ErrInvalidAccount
}

func (this *User) GetRoles() (rols roles.Roles) {
	return roles.NewRoles(this.Roles.Names()...)
}

func (this *User) GetAormInlinePreloadFields() []string {
	return []string{"Name", "Email"}
}

func (this *User) GetDefaultLocale() string {
	return this.Locale
}

func (this *User) GetLocales() []string {
	return []string{this.Locale}
}

func (this *User) GetTimeLocation() *time.Location {
	return time.Local
}

func (this *User) SystemAdmin() {
	this.Super = true
	this.Name = "admin"
	this.Confirmed = true
	this.Super = true
	if this.Email == "" {
		this.Email = "admin@localhost.com"
	}
}

func (this *User) String() string {
	return this.Name
}

func (this *User) GetName() string {
	return this.Name
}

func (this *User) DisplayName() string {
	return this.Email
}

func (this *User) GetEmail() string {
	return this.Email
}

func (this *User) AvailableLocales() []string {
	return []string{"pt-BR"}
}

func (this *User) Validate(db *aorm.DB) {
	if !this.Super && this.Name == "admin" {
		db.AddError(validations.Failed(this, "Name", "Invalid name."))
	}
}

func (this *User) Schema() *auth.Schema {
	return &auth.Schema{
		UID:      this.GetUID(),
		Name:     this.Name,
		Email:    this.Email,
		Location: this.Location,
		RawInfo:  this,
	}
}

type AvatarImageStorage struct{ oss.OSS }

func (AvatarImageStorage) GetSizes() map[string]*oss.Size {
	return map[string]*oss.Size{
		"small":  {Width: 50, Height: 50},
		"middle": {Width: 120, Height: 120},
		"big":    {Width: 320, Height: 320},
	}
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
	fragment.FragmentedModel

	Name, Description string

	Users []UserGroup `sql:"foreignkey:GroupID"`
}

type UserGroup struct {
	aorm.AuditedSDModel

	UserID  bid.BID `sql:"index;unique_index:ux_user_group"`
	User    *User
	GroupID bid.BID `sql:"index;unique_index:ux_user_group"`
	Group   *Group
}

func (UserGroup) GetAormInlinePreloadFields() []string {
	return []string{"*", "User"}
}
