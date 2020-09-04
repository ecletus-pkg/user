package user

import (
	"database/sql/driver"
	"reflect"
	"sort"
	"strings"

	"github.com/ecletus/admin"
	"github.com/ecletus/roles"
	"github.com/moisespsena-go/aorm"
)

type RolesAttribute struct {
	m map[string]bool
}

func (this RolesAttribute) AormDefaultDbValue(dialect aorm.Dialector) string {
	return dialect.ZeroValueOf(reflect.TypeOf(""))
}

func (this RolesAttribute) Names() (names []string) {
	if this.m != nil {
		for name := range this.m {
			names = append(names, name)
		}
	}
	return
}

func (this RolesAttribute) SortedNames() (names []string) {
	names = this.Names()
	sort.Strings(names)
	return
}

func (this *RolesAttribute) Scan(src interface{}) error {
	this.m = map[string]bool{}
	switch t := src.(type) {
	case string:
		for _, name := range strings.Split(t, "|") {
			this.m[name] = true
		}
	case []byte:
		return this.Scan(string(t))
	}
	return nil
}

func (this RolesAttribute) Value() (driver.Value, error) {
	return strings.Join(this.SortedNames(), "|"), nil
}

func (this RolesAttribute) Values(ctx *admin.Context) interface{} {
	return this.Roles(ctx).Descriptors()
}

func (this RolesAttribute) Roles(ctx *admin.Context) roles.Roles {
	return ctx.Site.Role().Roles().Intersection(this.Names())
}

func (this RolesAttribute) GetCollection(ctx *admin.Context) (options [][]string) {
	ctx.Site.Role().Descriptors().Each(func(d *roles.Descriptor) {
		options = append(options, []string{d.Name, d.Translate(ctx.GetI18nContext())})
	})
	sort.Slice(options, func(i, j int) bool {
		return options[i][1] < options[j][1]
	})
	return
}

func (this *RolesAttribute) StringsScan(src []string) error {
	this.m = map[string]bool{}
	for _, v := range src {
		if v != "" {
			this.m[v] = true
		}
	}
	return nil
}

func (this RolesAttribute) Strings(ctx *admin.Context) (items []string) {
	return this.Roles(ctx).Labels(ctx.GetI18nContext())
}

func (this RolesAttribute) String(ctx *admin.Context) string {
	return strings.Join(this.Strings(ctx), ", ")
}

