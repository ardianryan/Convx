const { eq, desc } = require('drizzle-orm');
const { db, schema } = require('../db');

function listRelays() {
  return db
    .select()
    .from(schema.proxyRelays)
    .orderBy(desc(schema.proxyRelays.createdAt))
    .all();
}

function getActiveRelay() {
  const records = db
    .select()
    .from(schema.proxyRelays)
    .where(eq(schema.proxyRelays.isActive, 1))
    .all();

  return records.length > 0 ? records[0] : null;
}

function addRelay(name, url, type = 'cloudflare') {
  const now = Date.now();
  // Deactivate all others first so this becomes the primary active one
  db.update(schema.proxyRelays).set({ isActive: 0 }).run();

  const insertResult = db
    .insert(schema.proxyRelays)
    .values({
      name,
      url,
      type,
      isActive: 1,
      createdAt: now,
    })
    .run();

  return {
    id: insertResult.lastInsertRowid,
    name,
    url,
    type,
    isActive: 1,
    createdAt: now,
  };
}

function setActiveRelay(id) {
  // Deactivate all
  db.update(schema.proxyRelays).set({ isActive: 0 }).run();
  // Activate selected
  db.update(schema.proxyRelays)
    .set({ isActive: 1 })
    .where(eq(schema.proxyRelays.id, id))
    .run();
}

function toggleRelay(id, isActive) {
  db.update(schema.proxyRelays)
    .set({ isActive: isActive ? 1 : 0 })
    .where(eq(schema.proxyRelays.id, id))
    .run();
}

function deleteRelay(id) {
  db.delete(schema.proxyRelays).where(eq(schema.proxyRelays.id, id)).run();
}

module.exports = {
  listRelays,
  getActiveRelay,
  addRelay,
  setActiveRelay,
  toggleRelay,
  deleteRelay,
};
