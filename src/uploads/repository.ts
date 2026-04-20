import type { Sql } from 'postgres';
import type { Upload } from '../types';
import { generateId } from '../utils/crypto';

export interface UploadsRepository {
  create(input: CreateUploadInput): Promise<Upload>;
  get(id: string): Promise<Upload | null>;
  updateStatus(id: string, status: Upload['status'], downloadUrl?: string): Promise<void>;
  listByUser(userId: string): Promise<Upload[]>;
}

export interface CreateUploadInput {
  filename: string;
  mimeType: string;
  size: number;
  storageKey: string;
  uploadedBy: string;
}

export class PostgresUploadsRepository implements UploadsRepository {
  constructor(private sql: Sql) {}

  async create(input: CreateUploadInput): Promise<Upload> {
    const id = generateId();
    
    const rows = await this.sql`
      INSERT INTO uploads (
        id, filename, mime_type, size, storage_key, status, uploaded_by, created_at, updated_at
      ) VALUES (
        ${id}, ${input.filename}, ${input.mimeType}, ${input.size},
        ${input.storageKey}, 'pending', ${input.uploadedBy}, NOW(), NOW()
      ) RETURNING *
    `;
    
    return this.mapRowToUpload(rows[0]);
  }

  async get(id: string): Promise<Upload | null> {
    const rows = await this.sql`
      SELECT * FROM uploads WHERE id = ${id}
    `;
    
    if (rows.length === 0) return null;
    return this.mapRowToUpload(rows[0]);
  }

  async updateStatus(id: string, status: Upload['status'], downloadUrl?: string): Promise<void> {
    await this.sql`
      UPDATE uploads SET
        status = ${status},
        download_url = ${downloadUrl || null},
        updated_at = NOW()
      WHERE id = ${id}
    `;
  }

  async listByUser(userId: string): Promise<Upload[]> {
    const rows = await this.sql`
      SELECT * FROM uploads WHERE uploaded_by = ${userId} ORDER BY created_at DESC
    `;
    
    return rows.map(row => this.mapRowToUpload(row));
  }

  private mapRowToUpload(row: any): Upload {
    return {
      id: row.id,
      filename: row.filename,
      mimeType: row.mime_type,
      size: row.size,
      storageKey: row.storage_key,
      downloadUrl: row.download_url,
      status: row.status,
      uploadedBy: row.uploaded_by,
      createdAt: row.created_at,
      updatedAt: row.updated_at,
    };
  }
}
