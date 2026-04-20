import { Elysia, t } from 'elysia';
import type { PostgresAuthService } from './service';
import type { PostgresAuthRepository } from './repository';
import { signupSchema, loginSchema, updateProfileSchema } from '../utils/validation';

export function authRoutes(
  authService: PostgresAuthService,
  authRepository: PostgresAuthRepository
) {
  return new Elysia({ prefix: '/auth' })
    .post(
      '/signup',
      async ({ body }) => {
        try {
          const { user, token } = await authService.signup(body);
          return {
            user: {
              id: user.id,
              qkId: user.qkId,
              email: user.email,
              firstName: user.firstName,
              lastName: user.lastName,
              role: user.role,
              createdAt: user.createdAt,
            },
            token,
          };
        } catch (error) {
          if (error instanceof Error && error.message === 'QKID already in use') {
            throw new Error('QKID or email already in use');
          }
          throw new Error('Could not create user');
        }
      },
      { body: signupSchema }
    )
    .post(
      '/login',
      async ({ body }) => {
        try {
          const { user, token } = await authService.login(body.qkId, body.password);
          return {
            user: {
              id: user.id,
              qkId: user.qkId,
              email: user.email,
              firstName: user.firstName,
              lastName: user.lastName,
              role: user.role,
              createdAt: user.createdAt,
            },
            token,
          };
        } catch (error) {
          if (error instanceof Error && error.message === 'Invalid credentials') {
            throw new Error('Invalid QKID or password');
          }
          throw new Error('Could not log in');
        }
      },
      { body: loginSchema }
    )
    .get(
      '/me',
      async ({ cookie: { access_token } }) => {
        if (!access_token || typeof access_token !== 'string') {
          throw new Error('Unauthorized');
        }

        const payload = await authService.validateToken(access_token);
        if (!payload) {
          throw new Error('Unauthorized');
        }

        const user = await authRepository.getUserByID(payload.sub);
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
      },
      {
        cookie: t.Cookie({
          access_token: t.Optional(t.String()),
        }),
      }
    )
    .put(
      '/me/profile',
      async ({ body, cookie: { access_token } }) => {
        if (!access_token || typeof access_token !== 'string') {
          throw new Error('Unauthorized');
        }

        const payload = await authService.validateToken(access_token);
        if (!payload) {
          throw new Error('Unauthorized');
        }

        const user = await authRepository.updateUserProfile(
          payload.sub,
          body
        );

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
      },
      {
        body: updateProfileSchema,
        cookie: t.Cookie({
          access_token: t.Optional(t.String()),
        }),
      }
    )
    .get('/availability', async ({ query }) => {
      const { qkId } = query;
      
      if (!qkId || typeof qkId !== 'string') {
        throw new Error('QKID is required');
      }

      try {
        const user = await authRepository.getUserByQKID(qkId);
        return {
          qkId,
          available: !user,
        };
      } catch (error) {
        throw new Error('Could not check QKID availability');
      }
    }, {
      query: t.Object({
        qkId: t.String(),
      }),
    });
}
