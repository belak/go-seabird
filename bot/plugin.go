package bot

type Plugin interface {
	Reload(b *Bot) error
}

type PluginFactory func(b *Bot) (Plugin, error)

type AuthPlugin interface {
	Reload(b *Bot) error
	CheckPerm(nick string, perm string) bool
}

type AuthPluginFactory func(b *Bot) (AuthPlugin, error)
