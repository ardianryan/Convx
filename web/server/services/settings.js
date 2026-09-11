const { eq } = require('drizzle-orm');
const { db, schema } = require('../db');

function getSetting(key, defaultValue = null) {
  const records = db
    .select()
    .from(schema.settings)
    .where(eq(schema.settings.key, key))
    .all();

  if (records && records.length > 0) {
    return records[0].value;
  }
  return defaultValue;
}

function setSetting(key, value) {
  const strVal = String(value);
  const existing = db
    .select()
    .from(schema.settings)
    .where(eq(schema.settings.key, key))
    .all();

  if (existing && existing.length > 0) {
    db.update(schema.settings)
      .set({ value: strVal })
      .where(eq(schema.settings.key, key))
      .run();
  } else {
    db.insert(schema.settings)
      .values({ key, value: strVal })
      .run();
  }
}

function isInitialized() {
  const val = getSetting('is_initialized', 'false');
  return val === 'true';
}

function getPlatformName() {
  return getSetting('platform_name', 'Convx Music');
}

module.exports = {
  getSetting,
  setSetting,
  isInitialized,
  getPlatformName,
};
