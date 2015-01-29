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

CREATE TABLE IF NOT EXISTS forecast_location (
	nick VARCHAR(512),
	address VARCHAR(200),
	lat FLOAT,
	lon FLOAT,
	UNIQUE(nick)
);
