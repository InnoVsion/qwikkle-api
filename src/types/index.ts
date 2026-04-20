export type UserRole = 'user' | 'admin' | 'editor';
export type AccountStatus = 'active' | 'inactive' | 'suspended';
export type UploadStatus = 'pending' | 'completed' | 'failed';

export interface User {
  id: string;
  qkId: string;
  email?: string;
  passwordHash: string;
  firstName?: string;
  lastName?: string;
  phone?: string;
  gender?: string;
  dateOfBirth?: Date;
  country?: string;
  interests: string[];
  avatarUrl?: string;
  avatarStorageKey?: string;
  avatarDownloadUrl?: string;
  role: UserRole;
  status: AccountStatus;
  createdAt: Date;
  updatedAt: Date;
  lastLoginAt?: Date;
}

export interface Session {
  id: string;
  userId: string;
  refreshTokenHash: string;
  expiresAt: Date;
  createdAt: Date;
  revokedAt?: Date;
  userAgent?: string;
  ipAddress?: string;
}

export interface Upload {
  id: string;
  filename: string;
  mimeType: string;
  size: number;
  storageKey: string;
  downloadUrl?: string;
  status: UploadStatus;
  uploadedBy: string;
  createdAt: Date;
  updatedAt: Date;
}

export interface Organization {
  id: string;
  name: string;
  description?: string;
  createdBy: string;
  createdAt: Date;
  updatedAt: Date;
}

export interface OrganizationMember {
  id: string;
  organizationId: string;
  userId: string;
  role: 'owner' | 'admin' | 'member';
  joinedAt: Date;
}

export interface OrganizationDocument {
  id: string;
  organizationId: string;
  title: string;
  content?: string;
  uploadId?: string;
  createdBy: string;
  createdAt: Date;
  updatedAt: Date;
}

export interface JWTPayload {
  sub: string;
  qkId: string;
  role: UserRole;
  iat: number;
  exp: number;
  [key: string]: any; // Allow additional properties for JOSE compatibility
}

export interface SignupInput {
  qkId: string;
  email: string | undefined;
  password: string;
  firstName: string | undefined;
  lastName: string | undefined;
  phone: string | undefined;
  gender: string | undefined;
  dateOfBirth: Date | undefined;
  country: string | undefined;
  interests: string[];
  avatarUploadId: string | undefined;
}

export interface LoginInput {
  qkId: string;
  password: string;
}

export interface UpdateUserProfileInput {
  email: string | undefined;
  firstName: string | undefined;
  lastName: string | undefined;
  phone: string | undefined;
  gender: string | undefined;
  dateOfBirth: Date | undefined;
  country: string | undefined;
  interests: string[] | undefined;
  avatarUploadId: string | undefined;
}
