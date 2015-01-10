CREATE TABLE IF NOT EXISTS karma (
	id SERIAL PRIMARY KEY,
	name VARCHAR(512) UNIQUE,
	score INTEGER
);

CREATE TABLE IF NOT EXISTS lastseen (
	name VARCHAR(512),
	channel VARCHAR(100),
	lastseen INTEGER,
	UNIQUE(name, channel)
);

CREATE TABLE IF NOT EXISTS nicks (
	nick VARCHAR(512),
	channel VARCHAR(100),
	flags VARCHAR(50),
	UNIQUE(nick, channel)
);

CREATE TABLE IF NOT EXISTS keystore (
	channel VARCHAR(100),
	key VARCHAR(50),
	value VARCHAR(50),
	UNIQUE(channel, key)
);
