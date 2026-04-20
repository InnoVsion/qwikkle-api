import { S3Client, PutObjectCommand, GetObjectCommand } from '@aws-sdk/client-s3';
import { getSignedUrl } from '@aws-sdk/s3-request-presigner';

export interface S3Config {
  region: string;
  endpoint?: string | undefined;
  accessKeyId: string;
  secretAccessKey: string;
}

export interface Presigner {
  getUploadUrl(key: string, contentType: string): Promise<string>;
  getDownloadUrl(key: string): Promise<string>;
}

export class S3Presigner implements Presigner {
  private client: S3Client;
  private bucket: string;

  constructor(config: S3Config, bucket: string) {
    const clientConfig: any = {
      region: config.region,
      credentials: {
        accessKeyId: config.accessKeyId,
        secretAccessKey: config.secretAccessKey,
      },
    };
    
    if (config.endpoint) {
      clientConfig.endpoint = config.endpoint;
    }
    
    this.client = new S3Client(clientConfig);
    this.bucket = bucket;
  }

  async getUploadUrl(key: string, contentType: string): Promise<string> {
    const command = new PutObjectCommand({
      Bucket: this.bucket,
      Key: key,
      ContentType: contentType,
    });

    return await getSignedUrl(this.client, command, { expiresIn: 3600 });
  }

  async getDownloadUrl(key: string): Promise<string> {
    const command = new GetObjectCommand({
      Bucket: this.bucket,
      Key: key,
    });

    return await getSignedUrl(this.client, command, { expiresIn: 3600 });
  }
}

export class NoopPresigner implements Presigner {
  async getUploadUrl(_key: string, _contentType: string): Promise<string> {
    throw new Error('S3 not configured');
  }

  async getDownloadUrl(_key: string): Promise<string> {
    throw new Error('S3 not configured');
  }
}
