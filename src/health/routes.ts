import { Elysia } from 'elysia';
import type { Database } from '../db';

export function healthRoutes(db: Database) {
  return new Elysia({ prefix: '/health' })
    .get('/', async () => {
      return {
        status: 'ok',
        timestamp: new Date().toISOString(),
      };
    })
    .get('/readyz', async () => {
      try {
        await db.ping();
        
        // Check if required tables exist
        const tables = [
          'goose_db_version',
          'users',
          'sessions',
          'organizations',
          'organization_members',
          'organization_documents',
          'uploads',
        ];

        const tableChecks = await Promise.all(
          tables.map(async (name) => {
            const result = await db.sql`
              SELECT to_regclass('public.${name}') IS NOT NULL as exists
            `;
            return {
              name,
              exists: result[0]?.exists || false,
            };
          })
        );

        const allTablesExist = tableChecks.every(check => check.exists);
        const status = allTablesExist ? 'ok' : 'degraded';
        const statusCode = allTablesExist ? 200 : 503;

        return {
          status,
          db: 'ok',
          tables: tableChecks,
        };
      } catch (error) {
        return {
          status: 'error',
          db: 'down',
          error: error instanceof Error ? error.message : 'Unknown error',
        };
      }
    });
}
