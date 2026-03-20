import pkg from 'pg';
const { Client } = pkg;
import dotenv from 'dotenv';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// Resolve .env.local from the api package root, independent of current working directory.
dotenv.config({path: path.resolve(__dirname, '../.env.local')});

const connectionConfig = {
  user: process.env.PG_USER || process.env.PGUSER || process.env.USER,
  host: process.env.PG_HOST || process.env.PGHOST || 'localhost',
  database: process.env.PG_DB || process.env.PGDATABASE || 'readpanda',
  password: process.env.PG_PASSWORD || process.env.PGPASSWORD || '',
  port: Number(process.env.PG_PORT || process.env.PGPORT || 5432),
};

const client = new Client(connectionConfig);

// Export the client instance for use in other parts of the application
export default client;
