import { getPluginApiUrl } from './pluginUrl';

export type ExportFormat = 'json' | 'markdown';

export async function exportConversation(
  conversationId: string,
  format: ExportFormat
): Promise<void> {
  const url = getPluginApiUrl(`/conversations/${conversationId}/export?format=${format}`);

  try {
    const response = await fetch(url, {
      method: 'GET',
      credentials: 'same-origin',
    });

    if (!response.ok) {
      const errorText = await response.text();
      throw new Error(`Export failed: ${response.status} - ${errorText}`);
    }

    const disposition = response.headers.get('Content-Disposition');
    const filename = extractFilename(disposition, conversationId, format);
    const blob = await response.blob();

    downloadBlob(blob, filename);
  } catch (error) {
    console.error('Export failed', error);
    throw error;
  }
}

function extractFilename(
  disposition: string | null,
  conversationId: string,
  format: ExportFormat
): string {
  if (disposition) {
    const filenameMatch = disposition.match(/filename="(.+)"/);
    if (filenameMatch?.[1]) {
      return filenameMatch[1];
    }

    const filenameStarMatch = disposition.match(/filename=([^;]+)/);
    if (filenameStarMatch?.[1]) {
      return filenameStarMatch[1].trim();
    }
  }

  const timestamp = new Date().toISOString().replace(/[:.]/g, '-').substring(0, 19);
  const extension = format === 'json' ? 'json' : 'md';
  return `zagalin-conversation-${conversationId}-${timestamp}.${extension}`;
}

function downloadBlob(blob: Blob, filename: string): void {
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = filename;
  link.style.display = 'none';

  document.body.appendChild(link);
  link.click();

  setTimeout(() => {
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
  }, 100);
}
