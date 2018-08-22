package user

import (
	"github.com/aghape/core"
	"github.com/aghape/plug"
)

var E_REGISTER_USER = PKG + ".register_user"

type RegisterUserEvent struct {
	plug.PluginEventInterface
	Site core.SiteInterface
}

type events struct {
	d plug.EventDispatcherInterface
}

func (e *events) OnRegisterUsers(siteName string, cb func(e *RegisterUserEvent) error) {
	e.d.On(E_REGISTER_USER+":"+siteName, func(e plug.PluginEventInterface) error {
		return cb(e.(*RegisterUserEvent))
	})
}

func Events(d plug.PluginEventDispatcherInterface) *events {
	return &events{d}
}

type trigger struct {
	d plug.EventDispatcherInterface
}

func Trigger(d plug.PluginEventDispatcherInterface) *trigger {
	return &trigger{d}
}

func (d *trigger) RegisterUsers(site core.SiteInterface) error {
	return d.d.Trigger(&RegisterUserEvent{plug.NewPluginEvent(E_REGISTER_USER + ":" + site.Name()), site})
}
