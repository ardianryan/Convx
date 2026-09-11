const path = require('path');
const fs = require('fs');
const Database = require('better-sqlite3');
const { drizzle } = require('drizzle-orm/better-sqlite3');
const schema = require('./schema');

// Ensure data directory exists
const dbDir = path.resolve(__dirname, '../../backend/data');
if (!fs.existsSync(dbDir)) {
  fs.mkdirSync(dbDir, { recursive: true });
}

const dbPath = path.join(dbDir, 'convx.db');
const sqlite = new Database(dbPath);
sqlite.pragma('journal_mode = WAL');

// Initialize tables if they don't exist
sqlite.exec(`
  CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
  );

  CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at INTEGER NOT NULL
  );

  CREATE TABLE IF NOT EXISTS sessions (
    token TEXT PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at INTEGER NOT NULL,
    created_at INTEGER NOT NULL
  );

  CREATE TABLE IF NOT EXISTS proxy_relays (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    type TEXT NOT NULL DEFAULT 'cloudflare',
    url TEXT NOT NULL,
    is_active INTEGER NOT NULL DEFAULT 1,
    created_at INTEGER NOT NULL
  );

  CREATE TABLE IF NOT EXISTS devices (
    id TEXT PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    device_type TEXT NOT NULL,
    platform TEXT NOT NULL,
    browser TEXT NOT NULL,
    is_playing INTEGER NOT NULL DEFAULT 0,
    current_song TEXT,
    last_heartbeat INTEGER NOT NULL,
    created_at INTEGER NOT NULL
  );
`);

const db = drizzle(sqlite, { schema });

module.exports = {
  db,
  sqlite,
  schema,
};
