package user

import "github.com/ecletus/admin"

const (
	ActionLogout     = "Logout"
	ActionBulkLogout = "BulkLogout"
)

type Logouter interface {
	Logout(user interface{}, context *admin.Context)
}

type Logouters []Logouter

func (logouters *Logouters) Append(logouter ...Logouter) {
	*logouters = append(*logouters, logouter...)
}
