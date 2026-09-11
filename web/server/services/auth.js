const crypto = require('crypto');
const bcrypt = require('bcryptjs');
const { eq } = require('drizzle-orm');
const { db, schema } = require('../db');

const SESSION_TTL_MS = 30 * 24 * 60 * 60 * 1000; // 30 days

async function hashPassword(plainPassword) {
  return await bcrypt.hash(plainPassword, 10);
}

async function verifyPassword(plainPassword, hash) {
  return await bcrypt.compare(plainPassword, hash);
}

async function createSession(userId) {
  const token = crypto.randomBytes(32).toString('hex');
  const now = Date.now();
  const expiresAt = now + SESSION_TTL_MS;

  db.insert(schema.sessions)
    .values({
      token,
      userId,
      expiresAt,
      createdAt: now,
    })
    .run();

  return { token, expiresAt };
}

async function validateSession(token) {
  if (!token) return null;

  const now = Date.now();
  const sessionRecords = db
    .select()
    .from(schema.sessions)
    .where(eq(schema.sessions.token, token))
    .all();

  if (!sessionRecords || sessionRecords.length === 0) {
    return null;
  }

  const session = sessionRecords[0];
  if (session.expiresAt < now) {
    // Expired
    db.delete(schema.sessions).where(eq(schema.sessions.token, token)).run();
    return null;
  }

  // Get user
  const userRecords = db
    .select({
      id: schema.users.id,
      username: schema.users.username,
      createdAt: schema.users.createdAt,
    })
    .from(schema.users)
    .where(eq(schema.users.id, session.userId))
    .all();

  if (!userRecords || userRecords.length === 0) {
    return null;
  }

  return {
    user: userRecords[0],
    session,
  };
}

async function destroySession(token) {
  if (!token) return;
  db.delete(schema.sessions).where(eq(schema.sessions.token, token)).run();
}

module.exports = {
  hashPassword,
  verifyPassword,
  createSession,
  validateSession,
  destroySession,
};
