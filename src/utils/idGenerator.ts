export function generateArtifactId(): string {
  return `art_${Date.now()}_${Math.random().toString(36).substring(2, 9)}`;
}

export function generateMessageId(): string {
  return `${Date.now()}-${Math.random().toString(36).substring(2, 11)}`;
}
