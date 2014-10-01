package bot

type Reloader interface {
	Reload(b *Bot) error
}

type Plugin interface{}
type PluginFactory func(b *Bot) (Plugin, error)

type AuthPlugin interface {
	CheckPerm(nick string, perm string) bool
}
type AuthPluginFactory func(b *Bot) (AuthPlugin, error)
