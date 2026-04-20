import { SignJWT, jwtVerify } from 'jose';
import type { User, SignupInput, JWTPayload } from '../types';
import type { AuthRepository } from './repository';
import { generateRefreshToken, hashToken } from '../utils/crypto';
import { normalizeQKID } from '../utils/validation';

export interface AuthService {
  signup(input: SignupInput): Promise<{ user: User; token: string }>;
  login(qkId: string, password: string): Promise<{ user: User; token: string }>;
  generateAccessToken(user: User, expiresIn: number): Promise<string>;
  generateRefreshToken(): Promise<{ token: string; hash: string }>;
  validateToken(token: string): Promise<JWTPayload | null>;
}

export class PostgresAuthService implements AuthService {
  constructor(
    private repository: AuthRepository,
    private jwtSecret: string
  ) {}

  async signup(input: SignupInput): Promise<{ user: User; token: string }> {
    const normalizedQKID = normalizeQKID(input.qkId);
    
    // Check if QKID or email already exists
    const existingUser = await this.repository.getUserByQKID(normalizedQKID);
    if (existingUser) {
      throw new Error('QKID already in use');
    }

    if (input.email) {
      // TODO: Check email uniqueness if needed
    }

    const user = await this.repository.createUser(input);
    const token = await this.generateAccessToken(user, 24 * 60 * 60); // 24 hours

    return { user, token };
  }

  async login(qkId: string, password: string): Promise<{ user: User; token: string }> {
    const normalizedQKID = normalizeQKID(qkId);
    const user = await this.repository.getUserByQKID(normalizedQKID);
    
    if (!user) {
      throw new Error('Invalid credentials');
    }

    // TODO: Verify password - need to import verifyPassword
    const { verifyPassword } = await import('../utils/crypto');
    const isValidPassword = await verifyPassword(password, user.passwordHash);
    
    if (!isValidPassword) {
      throw new Error('Invalid credentials');
    }

    // Update last login
    // TODO: Add method to update last login

    const token = await this.generateAccessToken(user, 24 * 60 * 60); // 24 hours

    return { user, token };
  }

  async generateAccessToken(user: User, expiresIn: number): Promise<string> {
    const now = Math.floor(Date.now() / 1000);
    const payload: JWTPayload = {
      sub: user.id,
      qkId: user.qkId,
      role: user.role,
      iat: now,
      exp: now + expiresIn,
    };

    const secret = new TextEncoder().encode(this.jwtSecret);
    return await new SignJWT(payload)
      .setProtectedHeader({ alg: 'HS256' })
      .setIssuedAt(payload.iat)
      .setExpirationTime(payload.exp)
      .sign(secret);
  }

  async generateRefreshToken(): Promise<{ token: string; hash: string }> {
    const token = generateRefreshToken();
    const hash = await hashToken(token);
    return { token, hash };
  }

  async validateToken(token: string): Promise<JWTPayload | null> {
    try {
      const secret = new TextEncoder().encode(this.jwtSecret);
      const { payload } = await jwtVerify(token, secret);
      return payload as JWTPayload;
    } catch {
      return null;
    }
  }
}
