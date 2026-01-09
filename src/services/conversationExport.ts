import type { Conversation } from './conversationStorage';

export type ExportFormat = 'json' | 'markdown';

/**
 * Export conversation directly from localStorage (client-side)
 * No backend API call needed since conversations are stored in localStorage
 */
export async function exportConversation(
  conversation: Conversation,
  format: ExportFormat
): Promise<void> {
  try {
    const content = format === 'json'
      ? exportAsJSON(conversation)
      : exportAsMarkdown(conversation);

    const blob = new Blob([content], {
      type: format === 'json' ? 'application/json' : 'text/markdown'
    });

    const filename = generateFilename(conversation, format);
    downloadBlob(blob, filename);
  } catch (error) {
    console.error('Export failed', error);
    throw error;
  }
}

function exportAsJSON(conversation: Conversation): string {
  return JSON.stringify(conversation, null, 2);
}

function exportAsMarkdown(conversation: Conversation): string {
  const lines: string[] = [];

  lines.push(`# ${conversation.title}\n`);
  lines.push(`**Created:** ${new Date(conversation.createdAt).toLocaleString()}`);
  lines.push(`**Updated:** ${new Date(conversation.updatedAt).toLocaleString()}`);
  lines.push(`**Messages:** ${conversation.messages.length}\n`);

  if (conversation.contexts && conversation.contexts.length > 0) {
    lines.push(`## Context\n`);
    conversation.contexts.forEach((ctx, idx) => {
      lines.push(`### Dashboard ${idx + 1}: ${ctx.dashboardTitle}`);
      lines.push(`- **UID:** ${ctx.dashboardUid}`);
      if (ctx.panelId) {
        lines.push(`- **Panel:** ${ctx.panelTitle || ctx.panelId}`);
      }
      if (ctx.timeFrom && ctx.timeTo) {
        lines.push(`- **Time Range:** ${ctx.timeFrom} to ${ctx.timeTo}`);
      }
      lines.push('');
    });
  }

  lines.push(`## Conversation\n`);

  conversation.messages.forEach((msg, idx) => {
    const timestamp = new Date(msg.timestamp).toLocaleString();
    const role = msg.role === 'user' ? '👤 User' : '🤖 Assistant';

    lines.push(`### ${role} - ${timestamp}\n`);
    lines.push(msg.content);
    lines.push('');

    if (msg.confidence !== undefined || msg.reasoning || msg.sources) {
      lines.push('**Metadata:**');
      if (msg.confidence !== undefined) {
        lines.push(`- Confidence: ${msg.confidence}%`);
      }
      if (msg.reasoning && msg.reasoning.length > 0) {
        lines.push(`- Reasoning steps: ${msg.reasoning.length}`);
      }
      if (msg.sources && msg.sources.length > 0) {
        lines.push(`- Sources: ${msg.sources.length}`);
      }
      if (msg.tokens !== undefined) {
        lines.push(`- Tokens: ${msg.tokens}`);
      }
      if (msg.cost !== undefined) {
        lines.push(`- Cost: $${msg.cost.toFixed(4)}`);
      }
      lines.push('');
    }

    lines.push('---\n');
  });

  lines.push(`\n*Exported from Zagalin on ${new Date().toLocaleString()}*`);

  return lines.join('\n');
}

function generateFilename(conversation: Conversation, format: ExportFormat): string {
  const timestamp = new Date().toISOString().replace(/[:.]/g, '-').substring(0, 19);
  const extension = format === 'json' ? 'json' : 'md';
  const safeTitle = conversation.title
    .replace(/[^a-z0-9]/gi, '-')
    .replace(/-+/g, '-')
    .substring(0, 50);

  return `zagalin-${safeTitle}-${timestamp}.${extension}`;
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
