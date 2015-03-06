package bot

// A plugin is fairly simple in that it only needs to have a method that will
// register itself with the muxes inside the bot that it needs, however the
// actual plugin can be as simple or as complex as needed.
type Plugin interface {
	Register(b *Bot) error
}
