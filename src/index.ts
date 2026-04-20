import { Elysia } from 'elysia';
import { cors } from '@elysiajs/cors';
import { swagger } from '@elysiajs/swagger';
import { loadConfig } from './config';
import { createDatabase } from './db';
import { healthRoutes } from './health/routes';
import { authRoutes } from './auth/routes';
import { uploadRoutes } from './uploads/routes';
import { adminRoutes } from './admin/routes';
import { PostgresAuthRepository } from './auth/repository';
import { PostgresAuthService } from './auth/service';
import { PostgresUploadsRepository } from './uploads/repository';
import { S3Presigner, NoopPresigner } from './storage/s3';

const config = loadConfig();
const db = createDatabase(config.postgresDsn);

// Initialize repositories and services
const authRepository = new PostgresAuthRepository(db.sql);
const authService = new PostgresAuthService(authRepository, config.jwtAccessSecret);
const uploadsRepository = new PostgresUploadsRepository(db.sql);

// Bootstrap admin user
await authRepository.bootstrapAdmin();

// Initialize S3 presigner if configured
let presigner = new NoopPresigner();
if (config.s3Bucket && config.s3AccessKeyId && config.s3SecretAccessKey) {
  presigner = new S3Presigner(
    {
      region: config.s3Region,
      endpoint: config.s3Endpoint || undefined,
      accessKeyId: config.s3AccessKeyId,
      secretAccessKey: config.s3SecretAccessKey,
    },
    config.s3Bucket
  );
  console.log('S3 presigner configured');
} else {
  console.warn('S3 not configured; uploads disabled');
}

const app = new Elysia()
  .use(
    cors({
      origin: config.corsAllowedOrigins.length > 0 ? config.corsAllowedOrigins : true,
      credentials: true,
      methods: ['GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'OPTIONS'],
      allowedHeaders: ['Content-Type', 'Authorization'],
      exposeHeaders: ['Set-Cookie'],
    })
  )
  .use(
    swagger({
      documentation: {
        info: {
          title: 'Qwikkle API',
          version: '1.0.0',
        },
        tags: [
          { name: 'Health', description: 'Health check endpoints' },
          { name: 'Auth', description: 'Authentication endpoints' },
          { name: 'Uploads', description: 'File upload endpoints' },
          { name: 'Admin', description: 'Admin endpoints' },
        ],
      },
    })
  )
  .use(healthRoutes(db))
  .use(authRoutes(authService, authRepository))
  .use(uploadRoutes(uploadsRepository, presigner))
  .use(adminRoutes(authRepository, uploadsRepository, presigner, config.jwtAccessSecret))
  .get('/', () => ({
    name: 'Qwikkle API',
    version: '1.0.0',
    framework: 'ElysiaJS',
    timestamp: new Date().toISOString(),
  }))
  .listen(config.port);

console.log(`🚀 Qwikkle API is running at http://localhost:${config.port}`);
console.log(`📚 Swagger docs available at http://localhost:${config.port}/swagger`);

// Graceful shutdown
process.on('SIGINT', async () => {
  console.log('\n🛑 Shutting down gracefully...');
  await db.close();
  process.exit(0);
});

process.on('SIGTERM', async () => {
  console.log('\n🛑 Shutting down gracefully...');
  await db.close();
  process.exit(0);
});

export default app;
