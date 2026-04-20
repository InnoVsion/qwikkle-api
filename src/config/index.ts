import { config } from 'dotenv';

// Load environment variables
config();

export interface AppConfig {
  port: number;
  appEnv: string;
  jwtAccessSecret: string;
  jwtRefreshSecret: string;
  cookieDomain: string;
  corsAllowedOrigins: string[];
  postgresDsn: string;
  s3Region: string;
  s3Bucket: string;
  s3Endpoint: string;
  s3AccessKeyId: string;
  s3SecretAccessKey: string;
}

export function loadConfig(): AppConfig {
  const port = Number(process.env.PORT) || 8080;
  const appEnv = process.env.APP_ENV || 'local';
  
  const jwtAccessSecret = process.env.JWT_ACCESS_SECRET;
  if (!jwtAccessSecret) {
    throw new Error('JWT_ACCESS_SECRET must be set');
  }
  
  const jwtRefreshSecret = process.env.JWT_REFRESH_SECRET;
  if (!jwtRefreshSecret) {
    throw new Error('JWT_REFRESH_SECRET must be set');
  }

  const postgresDsn = process.env.POSTGRES_DSN;
  if (!postgresDsn) {
    throw new Error('POSTGRES_DSN must be set');
  }

  const corsAllowedOrigins = process.env.CORS_ALLOWED_ORIGINS
    ? process.env.CORS_ALLOWED_ORIGINS.split(',').map(origin => origin.trim())
    : [];

  return {
    port,
    appEnv,
    jwtAccessSecret,
    jwtRefreshSecret,
    cookieDomain: process.env.COOKIE_DOMAIN || '',
    corsAllowedOrigins,
    postgresDsn,
    s3Region: process.env.S3_REGION || '',
    s3Bucket: process.env.S3_BUCKET || '',
    s3Endpoint: process.env.S3_ENDPOINT || '',
    s3AccessKeyId: process.env.S3_ACCESS_KEY_ID || '',
    s3SecretAccessKey: process.env.S3_SECRET_ACCESS_KEY || '',
  };
}
