import { Elysia, t } from 'elysia';
import type { PostgresUploadsRepository } from './repository';
import type { Presigner } from '../storage/s3';
import { generateId } from '../utils/crypto';

export function uploadRoutes(
  uploadsRepo: PostgresUploadsRepository,
  presigner: Presigner
) {
  return new Elysia({ prefix: '/uploads' })
    .post(
      '/presigned-url',
      async ({ body }) => {
        const { filename, mimeType, size } = body;
        
        // Validate file size (e.g., 10MB limit)
        if (size > 10 * 1024 * 1024) {
          throw new Error('File size too large (max 10MB)');
        }

        // Validate mime type
        const allowedTypes = [
          'image/jpeg',
          'image/png',
          'image/gif',
          'image/webp',
          'application/pdf',
          'text/plain',
        ];
        
        if (!allowedTypes.includes(mimeType)) {
          throw new Error('File type not allowed');
        }

        const uploadId = generateId();
        const storageKey = `uploads/${uploadId}/${filename}`;
        
        // Create upload record
        await uploadsRepo.create({
          filename,
          mimeType,
          size,
          storageKey,
          uploadedBy: 'anonymous', // TODO: Get from auth context
        });

        // Generate presigned URL
        const uploadUrl = await presigner.getUploadUrl(storageKey, mimeType);

        return {
          uploadId,
          uploadUrl,
          storageKey,
        };
      },
      {
        body: t.Object({
          filename: t.String(),
          mimeType: t.String(),
          size: t.Number(),
        }),
      }
    )
    .post(
      '/:uploadId/complete',
      async ({ params }) => {
        const { uploadId } = params;
        
        const upload = await uploadsRepo.get(uploadId);
        if (!upload) {
          throw new Error('Upload not found');
        }

        if (upload.status !== 'pending') {
          throw new Error('Upload already processed');
        }

        // Generate download URL
        const downloadUrl = await presigner.getDownloadUrl(upload.storageKey);
        
        // Mark upload as completed
        await uploadsRepo.updateStatus(uploadId, 'completed', downloadUrl);

        return {
          uploadId,
          status: 'completed',
          downloadUrl,
        };
      },
      {
        params: t.Object({
          uploadId: t.String(),
        }),
      }
    )
    .get(
      '/:uploadId',
      async ({ params }) => {
        const { uploadId } = params;
        
        const upload = await uploadsRepo.get(uploadId);
        if (!upload) {
          throw new Error('Upload not found');
        }

        return {
          upload: {
            id: upload.id,
            filename: upload.filename,
            mimeType: upload.mimeType,
            size: upload.size,
            status: upload.status,
            downloadUrl: upload.downloadUrl,
            createdAt: upload.createdAt,
          },
        };
      },
      {
        params: t.Object({
          uploadId: t.String(),
        }),
      }
    );
}
