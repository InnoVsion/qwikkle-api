import { Elysia, t } from 'elysia';
import type { PostgresAuthRepository } from '../auth/repository';
import type { Presigner } from '../storage/s3';
import type { PostgresUploadsRepository } from '../uploads/repository';

export function adminRoutes(
  authRepository: PostgresAuthRepository,
  _uploadsRepo: PostgresUploadsRepository,
  _presigner: Presigner,
  jwtSecret: string
) {
  return new Elysia({ prefix: '/admin' })
    .derive(async ({ headers }) => {
      const authHeader = headers.authorization;
      if (!authHeader?.startsWith('Bearer ')) {
        throw new Error('Unauthorized');
      }

      const token = authHeader.substring(7);
      const { PostgresAuthService } = await import('../auth/service');
      const authService = new PostgresAuthService(authRepository, jwtSecret);
      
      const payload = await authService.validateToken(token);
      if (!payload) {
        throw new Error('Unauthorized');
      }

      const user = await authRepository.getUserByID(payload.sub);
      if (!user || (user.role !== 'admin' && user.role !== 'editor')) {
        throw new Error('Forbidden');
      }

      return { user };
    })
    .get('/users', async ({ query }) => {
      const { page = '1', limit = '20' } = query;
      
      // TODO: Implement user listing with pagination and search
      return {
        users: [],
        pagination: {
          page: parseInt(page),
          limit: parseInt(limit),
          total: 0,
          totalPages: 0,
        },
      };
    }, {
      query: t.Object({
        page: t.Optional(t.String()),
        limit: t.Optional(t.String()),
        search: t.Optional(t.String()),
      }),
    })
    .get('/users/:userId', async ({ params }) => {
      const { userId } = params;
      
      const user = await authRepository.getUserByID(userId);
      if (!user) {
        throw new Error('User not found');
      }

      return {
        user: {
          id: user.id,
          qkId: user.qkId,
          email: user.email,
          firstName: user.firstName,
          lastName: user.lastName,
          phone: user.phone,
          gender: user.gender,
          dateOfBirth: user.dateOfBirth,
          country: user.country,
          interests: user.interests,
          avatarUrl: user.avatarUrl,
          role: user.role,
          status: user.status,
          createdAt: user.createdAt,
          updatedAt: user.updatedAt,
          lastLoginAt: user.lastLoginAt,
        },
      };
    }, {
      params: t.Object({
        userId: t.String(),
      }),
    })
    .patch('/users/:userId/status', async ({ params, body }) => {
      const { userId } = params;
      const { status } = body;
      
      const user = await authRepository.getUserByID(userId);
      if (!user) {
        throw new Error('User not found');
      }

      // TODO: Implement user status update
      return {
        user: {
          id: user.id,
          status,
        },
      };
    }, {
      params: t.Object({
        userId: t.String(),
      }),
      body: t.Object({
        status: t.Union([t.Literal('active'), t.Literal('inactive'), t.Literal('suspended')]),
      }),
    })
    .get('/uploads', async ({ query }) => {
      const { page = '1', limit = '20' } = query;
      
      // TODO: Implement uploads listing with pagination
      return {
        uploads: [],
        pagination: {
          page: parseInt(page),
          limit: parseInt(limit),
          total: 0,
          totalPages: 0,
        },
      };
    }, {
      query: t.Object({
        page: t.Optional(t.String()),
        limit: t.Optional(t.String()),
        status: t.Optional(t.Union([
          t.Literal('pending'),
          t.Literal('completed'),
          t.Literal('failed'),
        ])),
      }),
    })
    .get('/stats', async () => {
      // TODO: Implement admin statistics
      return {
        stats: {
          totalUsers: 0,
          activeUsers: 0,
          totalUploads: 0,
          pendingUploads: 0,
        },
      };
    });
}
