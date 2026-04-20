import { z } from 'zod';

export const normalizeQKID = (qkId: string): string => {
  return qkId.toLowerCase().trim();
};

export const qkIdSchema = z.string()
  .min(3, 'QKID must be at least 3 characters')
  .max(50, 'QKID must be at most 50 characters')
  .regex(/^[a-zA-Z0-9_-]+$/, 'QKID can only contain letters, numbers, hyphens, and underscores')
  .transform(normalizeQKID);

export const passwordSchema = z.string()
  .min(6, 'Password must be at least 6 characters')
  .max(100, 'Password must be at most 100 characters');

export const emailSchema = z.string().email('Invalid email address').optional();

export const signupSchema = z.object({
  qkId: qkIdSchema,
  email: emailSchema,
  password: passwordSchema,
  firstName: z.string().max(100).optional(),
  lastName: z.string().max(100).optional(),
  phone: z.string().max(20).optional(),
  gender: z.string().max(20).optional(),
  dateOfBirth: z.string().datetime().optional(),
  country: z.string().max(50).optional(),
  interests: z.array(z.string().max(50)).default([]),
  avatarUploadId: z.string().uuid().optional(),
});

export const loginSchema = z.object({
  qkId: qkIdSchema,
  password: z.string().min(1, 'Password is required'),
});

export const updateProfileSchema = z.object({
  email: emailSchema,
  firstName: z.string().max(100).optional(),
  lastName: z.string().max(100).optional(),
  phone: z.string().max(20).optional(),
  gender: z.string().max(20).optional(),
  dateOfBirth: z.string().datetime().optional(),
  country: z.string().max(50).optional(),
  interests: z.array(z.string().max(50)).optional(),
  avatarUploadId: z.string().uuid().optional(),
});
