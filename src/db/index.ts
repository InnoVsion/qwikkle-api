import postgres from 'postgres';
import type { Sql } from 'postgres';

export interface Database {
  sql: Sql;
  ping: () => Promise<void>;
  close: () => Promise<void>;
}

export function createDatabase(dsn: string): Database {
  const sql = postgres(dsn, {
    max: 10,
    idle_timeout: 20,
    connect_timeout: 10,
  });

  return {
    sql,
    ping: async () => {
      await sql`SELECT 1`;
    },
    close: async () => {
      await sql.end();
    },
  };
}

export type { Sql };
