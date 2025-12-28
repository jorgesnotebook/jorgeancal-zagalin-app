/**
 * Conversation Export Service - Download conversations as JSON or Markdown
 */

export type ExportFormat = 'json' | 'markdown';

/**
 * Export a conversation to a file and trigger download
 *
 * @param conversationId The conversation ID to export
 * @param format Export format (json or markdown)
 * @throws Error if export fails
 */
export async function exportConversation(
  conversationId: string,
  format: ExportFormat
): Promise<void> {
  const url = `/api/plugins/jorgeancal-zagalin-app/resources/conversations/${conversationId}/export?format=${format}`;

  try {
    const response = await fetch(url, {
      method: 'GET',
      credentials: 'same-origin',
    });

    if (!response.ok) {
      const errorText = await response.text();
      throw new Error(`Export failed: ${response.status} - ${errorText}`);
    }

    // Get filename from Content-Disposition header
    const disposition = response.headers.get('Content-Disposition');
    const filename = extractFilename(disposition, conversationId, format);

    // Download file
    const blob = await response.blob();
    downloadBlob(blob, filename);
  } catch (error) {
    console.error('Export failed', error);
    throw error;
  }
}

/**
 * Extract filename from Content-Disposition header
 *
 * @param disposition Content-Disposition header value
 * @param conversationId Fallback conversation ID
 * @param format Export format for extension
 * @returns Extracted or generated filename
 */
function extractFilename(
  disposition: string | null,
  conversationId: string,
  format: ExportFormat
): string {
  if (disposition) {
    // Try to extract filename from header
    const filenameMatch = disposition.match(/filename="(.+)"/);
    if (filenameMatch && filenameMatch[1]) {
      return filenameMatch[1];
    }

    // Try without quotes
    const filenameStarMatch = disposition.match(/filename=([^;]+)/);
    if (filenameStarMatch && filenameStarMatch[1]) {
      return filenameStarMatch[1].trim();
    }
  }

  // Fallback to generated filename
  const timestamp = new Date().toISOString().replace(/[:.]/g, '-').substring(0, 19);
  const extension = format === 'json' ? 'json' : 'md';
  return `zagalin-conversation-${conversationId}-${timestamp}.${extension}`;
}

/**
 * Trigger browser download of a blob
 *
 * @param blob The blob to download
 * @param filename The filename for the download
 */
function downloadBlob(blob: Blob, filename: string): void {
  // Create temporary download link
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = filename;
  link.style.display = 'none';

  // Append to body, click, and cleanup
  document.body.appendChild(link);
  link.click();

  // Cleanup after a short delay to ensure download starts
  setTimeout(() => {
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
  }, 100);
}
