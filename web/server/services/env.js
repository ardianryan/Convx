const fs = require('fs');
const path = require('path');

// Primary .env location in web/
const envPath = path.resolve(__dirname, '../../.env');

function loadEnv() {
  if (fs.existsSync(envPath)) {
    const content = fs.readFileSync(envPath, 'utf8');
    const lines = content.split('\n');
    for (const line of lines) {
      const trimmed = line.trim();
      if (trimmed && !trimmed.startsWith('#')) {
        const eqIdx = trimmed.indexOf('=');
        if (eqIdx !== -1) {
          const key = trimmed.slice(0, eqIdx).trim();
          const val = trimmed.slice(eqIdx + 1).trim();
          process.env[key] = val;
        }
      }
    }
  }
}

function updateEnv(updates) {
  loadEnv();
  let content = '';
  if (fs.existsSync(envPath)) {
    content = fs.readFileSync(envPath, 'utf8');
  }

  const lines = content.split('\n');
  const existingKeys = new Set();
  const newLines = [];

  for (const line of lines) {
    const trimmed = line.trim();
    if (trimmed && !trimmed.startsWith('#')) {
      const eqIdx = trimmed.indexOf('=');
      if (eqIdx !== -1) {
        const key = trimmed.slice(0, eqIdx).trim();
        if (key in updates) {
          newLines.push(`${key}=${updates[key]}`);
          existingKeys.add(key);
          process.env[key] = String(updates[key]);
          continue;
        }
      }
    }
    newLines.push(line);
  }

  // Append new keys that were not in .env before
  for (const [key, val] of Object.entries(updates)) {
    if (!existingKeys.has(key)) {
      newLines.push(`${key}=${val}`);
      process.env[key] = String(val);
    }
  }

  // Ensure trailing newline
  const finalContent = newLines.join('\n').replace(/\n+$/, '') + '\n';
  fs.writeFileSync(envPath, finalContent, 'utf8');

  // Also write to data/.env for backend if needed
  const dataEnvPath = path.resolve(__dirname, '../../backend/data/.env');
  try {
    fs.writeFileSync(dataEnvPath, finalContent, 'utf8');
  } catch (_) {}

  return process.env;
}

module.exports = {
  loadEnv,
  updateEnv,
  envPath,
};
