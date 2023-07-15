
use ccWs2;

CREATE TABLE tokens (
	`token`      CHAR(64) NOT NULL,
	`root`       BOOLEAN NOT NULL DEFAULT FALSE,
	`expiration` DATETIME, -- NULL if never expired
	PRIMARY KEY (`token`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE servers (
	`id` VARCHAR(64) NOT NULL,
	PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE token_ops (
	`token`  CHAR(64) NOT NULL,
	`server` VARCHAR(64) NOT NULL,
	PRIMARY KEY (`token`, `server`),
	CONSTRAINT op_token FOREIGN KEY (`token`)
	REFERENCES tokens(`token`) ON DELETE CASCADE ON UPDATE CASCADE,
	CONSTRAINT op_server FOREIGN KEY (`server`)
	REFERENCES servers(`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE daemon_tokens (
	`token`      CHAR(64) NOT NULL,
	`server`     VARCHAR(64) NOT NULL,
	`expiration` DATETIME, -- NULL if never expired
	PRIMARY KEY (`token`),
	CONSTRAINT daemon_server FOREIGN KEY (`server`)
	REFERENCES servers(`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE server_web_plugins (
	`server`  VARCHAR(64) NOT NULL,
	`plugin`  VARCHAR(64) NOT NULL,
	`version` VARCHAR(64) NOT NULL,
	PRIMARY KEY (`server`, `plugin`),
	CONSTRAINT server_web_plugin_sid FOREIGN KEY (`server`)
	REFERENCES servers(`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
