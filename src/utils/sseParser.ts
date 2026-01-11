import { Observable, Observer } from 'rxjs';

export interface SSEParseOptions<T> {
  onChunk: (chunk: T, observer: Observer<T>) => void;
  shouldComplete?: (chunk: T) => boolean;
}

export class SSEParser {
  static parseStream<T>(response: Response, options: SSEParseOptions<T>): Observable<T> {
    return new Observable<T>((observer) => {
      this.parse(response, observer, options).catch((error) => observer.error(error));
    });
  }

  private static async parse<T>(response: Response, observer: Observer<T>, options: SSEParseOptions<T>): Promise<void> {
    if (!response.body) {
      throw new Error('No response body');
    }

    const reader = response.body.getReader();
    const decoder = new TextDecoder();

    try {
      while (true) {
        const { done, value } = await reader.read();

        if (done) {
          observer.complete();
          break;
        }

        const text = decoder.decode(value, { stream: true });
        const lines = text.split('\n');

        for (const line of lines) {
          if (!line.trim() || !line.startsWith('data: ')) {
            continue;
          }

          const data = line.substring(6);

          if (data === '[DONE]') {
            observer.complete();
            return;
          }

          try {
            const chunk = JSON.parse(data) as T;
            options.onChunk(chunk, observer);

            if (options.shouldComplete && options.shouldComplete(chunk)) {
              observer.complete();
              return;
            }
          } catch (parseError) {
            console.warn('Failed to parse SSE chunk:', data, parseError);
          }
        }
      }
    } finally {
      reader.releaseLock();
    }
  }
}
