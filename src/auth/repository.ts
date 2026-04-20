import type { Sql } from 'postgres';
import type { User, Session, SignupInput, LoginInput, UpdateUserProfileInput } from '../types';
import { generateId, hashPassword, hashToken } from '../utils/crypto';
import { normalizeQKID } from '../utils/validation';

export interface AuthRepository {
  getUserByQKID(qkId: string): Promise<User | null>;
  getUserByID(id: string): Promise<User | null>;
  createUser(input: SignupInput): Promise<User>;
  updateUserProfile(id: string, input: UpdateUserProfileInput): Promise<User>;
  createSession(userId: string, refreshTokenHash: string, expiresAt: Date, userAgent?: string, ipAddress?: string): Promise<Session>;
  getSessionByRefreshTokenHash(refreshTokenHash: string): Promise<Session | null>;
  rotateSession(sessionId: string, newRefreshTokenHash: string, expiresAt: Date): Promise<void>;
  revokeSession(sessionId: string): Promise<void>;
  bootstrapAdmin(): Promise<void>;
}

export class PostgresAuthRepository implements AuthRepository {
  constructor(private sql: Sql) {}

  async getUserByQKID(qkId: string): Promise<User | null> {
    const normalizedQKID = normalizeQKID(qkId);
    const rows = await this.sql`
      SELECT * FROM users WHERE qk_id = ${normalizedQKID}
    `;
    
    if (rows.length === 0) return null;
    return this.mapRowToUser(rows[0]);
  }

  async getUserByID(id: string): Promise<User | null> {
    const rows = await this.sql`
      SELECT * FROM users WHERE id = ${id}
    `;
    
    if (rows.length === 0) return null;
    return this.mapRowToUser(rows[0]);
  }

  async createUser(input: SignupInput): Promise<User> {
    const id = generateId();
    const normalizedQKID = normalizeQKID(input.qkId);
    const passwordHash = await hashPassword(input.password);
    
    const rows = await this.sql`
      INSERT INTO users (
        id, qk_id, email, password_hash, first_name, last_name, phone,
        gender, date_of_birth, country, interests, avatar_url,
        avatar_storage_key, avatar_download_url, role, status, created_at, updated_at
      ) VALUES (
        ${id}, ${normalizedQKID}, ${input.email || null}, ${passwordHash},
        ${input.firstName || null}, ${input.lastName || null}, ${input.phone || null},
        ${input.gender || null}, ${input.dateOfBirth || null}, ${input.country || null},
        ${JSON.stringify(input.interests)}, ${null}, ${null}, ${null},
        'user', 'active', NOW(), NOW()
      ) RETURNING *
    `;
    
    return this.mapRowToUser(rows[0]);
  }

  async updateUserProfile(id: string, input: UpdateUserProfileInput): Promise<User> {
    const rows = await this.sql`
      UPDATE users SET
        email = ${input.email || null},
        first_name = ${input.firstName || null},
        last_name = ${input.lastName || null},
        phone = ${input.phone || null},
        gender = ${input.gender || null},
        date_of_birth = ${input.dateOfBirth || null},
        country = ${input.country || null},
        interests = ${input.interests ? JSON.stringify(input.interests) : null},
        updated_at = NOW()
      WHERE id = ${id}
      RETURNING *
    `;
    
    if (rows.length === 0) throw new Error('User not found');
    return this.mapRowToUser(rows[0]);
  }

  async createSession(
    userId: string,
    refreshTokenHash: string,
    expiresAt: Date,
    userAgent?: string,
    ipAddress?: string
  ): Promise<Session> {
    const id = generateId();
    
    const rows = await this.sql`
      INSERT INTO sessions (
        id, user_id, refresh_token_hash, expires_at, created_at,
        user_agent, ip_address
      ) VALUES (
        ${id}, ${userId}, ${refreshTokenHash}, ${expiresAt}, NOW(),
        ${userAgent || null}, ${ipAddress || null}
      ) RETURNING *
    `;
    
    return this.mapRowToSession(rows[0]);
  }

  async getSessionByRefreshTokenHash(refreshTokenHash: string): Promise<Session | null> {
    const rows = await this.sql`
      SELECT * FROM sessions WHERE refresh_token_hash = ${refreshTokenHash}
    `;
    
    if (rows.length === 0) return null;
    return this.mapRowToSession(rows[0]);
  }

  async rotateSession(sessionId: string, newRefreshTokenHash: string, expiresAt: Date): Promise<void> {
    await this.sql`
      UPDATE sessions SET
        refresh_token_hash = ${newRefreshTokenHash},
        expires_at = ${expiresAt},
        updated_at = NOW()
      WHERE id = ${sessionId}
    `;
  }

  async revokeSession(sessionId: string): Promise<void> {
    await this.sql`
      UPDATE sessions SET revoked_at = NOW() WHERE id = ${sessionId}
    `;
  }

  async bootstrapAdmin(): Promise<void> {
    const adminExists = await this.sql`
      SELECT 1 FROM users WHERE role = 'admin' LIMIT 1
    `;
    
    if (adminExists.length > 0) return;
    
    const adminId = generateId();
    const passwordHash = await hashPassword('admin123');
    
    await this.sql`
      INSERT INTO users (
        id, qk_id, email, password_hash, role, status, created_at, updated_at
      ) VALUES (
        ${adminId}, 'admin', 'admin@qwikkle.local', ${passwordHash},
        'admin', 'active', NOW(), NOW()
      )
    `;
  }

  private mapRowToUser(row: any): User {
    return {
      id: row.id,
      qkId: row.qk_id,
      email: row.email,
      passwordHash: row.password_hash,
      firstName: row.first_name,
      lastName: row.last_name,
      phone: row.phone,
      gender: row.gender,
      dateOfBirth: row.date_of_birth,
      country: row.country,
      interests: row.interests ? JSON.parse(row.interests) : [],
      avatarUrl: row.avatar_url,
      avatarStorageKey: row.avatar_storage_key,
      avatarDownloadUrl: row.avatar_download_url,
      role: row.role,
      status: row.status,
      createdAt: row.created_at,
      updatedAt: row.updated_at,
      lastLoginAt: row.last_login_at,
    };
  }

  private mapRowToSession(row: any): Session {
    return {
      id: row.id,
      userId: row.user_id,
      refreshTokenHash: row.refresh_token_hash,
      expiresAt: row.expires_at,
      createdAt: row.created_at,
      revokedAt: row.revoked_at,
      userAgent: row.user_agent,
      ipAddress: row.ip_address,
    };
  }
}
