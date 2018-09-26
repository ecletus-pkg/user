package user

import (
	"time"

	"github.com/aghape-pkg/people"
	"github.com/aghape/fragment"
	"github.com/aghape/media/oss"
	"github.com/aghape/validations"
	"github.com/moisespsena-go/aorm"
)

type User struct {
	aorm.AuditedModel
	fragment.FragmentedModel
	Email    string `form:"email"`
	Password string
	Name     string `form:"name"`
	Role     string

	// Confirm
	ConfirmToken string
	Confirmed    bool

	// Recover
	RecoverToken       string
	RecoverTokenExpiry *time.Time

	// Accepts
	AcceptPrivate bool `form:"accept-private"`
	AcceptLicense bool `form:"accept-license"`
	AcceptNews    bool `form:"accept-news"`
	systemAdmin   bool

	People people.People
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

type AvatarImageStorage struct{ oss.OSS }

func (AvatarImageStorage) GetSizes() map[string]*oss.Size {
	return map[string]*oss.Size{
		"small":  {Width: 50, Height: 50},
		"middle": {Width: 120, Height: 120},
		"big":    {Width: 320, Height: 320},
	}
}
