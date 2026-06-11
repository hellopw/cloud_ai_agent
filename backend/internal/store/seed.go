package store

import "time"

// SeedDefaultTools inserts default tools into the database if they don't already exist.
// This should be called after Migrate() to ensure default tools are available.
func (s *Store) SeedDefaultTools() error {
	defaults := []struct {
		name        string
		label       string
		description string
		dsl         string
	}{
		{
			name:        "mysql",
			label:       "MySQL MCP",
			description: "Connect to a MySQL database via MCP server. Set MYSQL_HOST, MYSQL_PORT, MYSQL_USER, MYSQL_PASSWORD, MYSQL_DATABASE environment variables.",
			dsl: `{
  "name": "mysql",
  "label": "MySQL MCP",
  "description": "Connect to a MySQL database via MCP server",
  "parameters": {},
  "handler": {
    "type": "mcp",
    "transport": "stdio",
    "command": "npx",
    "args": ["-y", "@benborber/mysql-mcp-server"],
    "env": {
      "MYSQL_HOST": "localhost",
      "MYSQL_PORT": "3306",
      "MYSQL_USER": "root",
      "MYSQL_PASSWORD": "",
      "MYSQL_DATABASE": ""
    }
  }
}`,
		},
		{
			name:        "postgres",
			label:       "PostgreSQL MCP",
			description: "Connect to a PostgreSQL database via MCP server. Set DATABASE_URL environment variable (e.g. postgresql://user:pass@localhost:5432/dbname).",
			dsl: `{
  "name": "postgres",
  "label": "PostgreSQL MCP",
  "description": "Connect to a PostgreSQL database via MCP server",
  "parameters": {},
  "handler": {
    "type": "mcp",
    "transport": "stdio",
    "command": "npx",
    "args": ["-y", "@modelcontextprotocol/server-postgres"],
    "env": {
      "DATABASE_URL": "postgresql://user:password@localhost:5432/dbname"
    }
  }
}`,
		},
		{
			name:        "sqlite",
			label:       "SQLite MCP",
			description: "Interact with a local SQLite database via MCP server. Set SQLITE_DB_PATH to the database file path.",
			dsl: `{
  "name": "sqlite",
  "label": "SQLite MCP",
  "description": "Interact with a local SQLite database via MCP server",
  "parameters": {},
  "handler": {
    "type": "mcp",
    "transport": "stdio",
    "command": "npx",
    "args": ["-y", "@nicholaschen/sqlite-mcp-server"],
    "env": {
      "SQLITE_DB_PATH": "./data.db"
    }
  }
}`,
		},
		{
			name:        "redis",
			label:       "Redis MCP",
			description: "Connect to a Redis instance via MCP server. Set REDIS_URL environment variable (e.g. redis://localhost:6379).",
			dsl: `{
  "name": "redis",
  "label": "Redis MCP",
  "description": "Connect to a Redis instance via MCP server",
  "parameters": {},
  "handler": {
    "type": "mcp",
    "transport": "stdio",
    "command": "npx",
    "args": ["-y", "@hxrxchang/redis-mcp-server"],
    "env": {
      "REDIS_URL": "redis://localhost:6379"
    }
  }
}`,
		},
		{
			name:        "mongodb",
			label:       "MongoDB MCP",
			description: "Connect to a MongoDB instance via MCP server. Set MONGODB_URI environment variable (e.g. mongodb://localhost:27017/dbname).",
			dsl: `{
  "name": "mongodb",
  "label": "MongoDB MCP",
  "description": "Connect to a MongoDB instance via MCP server",
  "parameters": {},
  "handler": {
    "type": "mcp",
    "transport": "stdio",
    "command": "npx",
    "args": ["-y", "@belencode/mongo-mcp-server"],
    "env": {
      "MONGODB_URI": "mongodb://localhost:27017/dbname"
    }
  }
}`,
		},
	}

	now := time.Now()
	for _, d := range defaults {
		var exists int
		if err := s.db.QueryRow("SELECT COUNT(*) FROM tools WHERE name = ?", d.name).Scan(&exists); err != nil {
			return err
		}
		if exists > 0 {
			continue
		}

		id := "default-" + d.name
		_, err := s.db.Exec(
			"INSERT OR IGNORE INTO tools (id, name, label, description, dsl_definition, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
			id, d.name, d.label, d.description, d.dsl, now, now,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

