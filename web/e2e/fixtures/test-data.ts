/**
 * Unique test data generators.
 * All identifiers include timestamps + random suffixes to prevent collisions across runs.
 */

import * as fs from 'fs';
import * as path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const ts = Date.now();
let counter = 0;

function uid(): string {
  counter++;
  return `${ts}-${counter}-${Math.random().toString(36).slice(2, 6)}`;
}

export function uniqueEmail(): string {
  return `e2e-${uid()}@test.local`;
}

export function uniqueName(prefix: string): string {
  return `${prefix}-${uid()}`.slice(0, 60);
}

export function uniqueSlug(prefix: string): string {
  return `${prefix}-${uid()}`.toLowerCase().replace(/[^a-z0-9-]/g, '').slice(0, 40);
}

export const TEST_PASSWORD = 'Test1234secure';

export const TEST_TEMPLATE_COMPOSE = `version: "3"
services:
  web:
    image: nginx:alpine
    ports:
      - "80:80"
`;

// Real cloud provider API keys for E2E tests — set via environment variable
// Export TEST_DO_API_KEY=dop_v1_... before running tests
export const TEST_DO_API_KEY = process.env.TEST_DO_API_KEY || '';

/**
 * Shared infrastructure state from global setup.
 * Contains real DigitalOcean droplet IDs, node, template, etc.
 */
export interface InfraState {
  token: string;
  email: string;
  nodeId: string;
  templateId: string;
  provisionId: string;
  credentialId: string;
  sshKeyId: string;
  dropletIp: string;
}

const INFRA_STATE_PATH = path.join(__dirname, '..', '.e2e-infra.json');

/**
 * Read the shared infrastructure state written by global-setup.ts.
 * Returns null if the state file doesn't exist (global setup didn't run).
 */
export function readInfraState(): InfraState | null {
  if (!fs.existsSync(INFRA_STATE_PATH)) return null;
  return JSON.parse(fs.readFileSync(INFRA_STATE_PATH, 'utf-8'));
}
