export interface HttpClient {
  fetch(url: string, init: RequestInit): Promise<Response>;
}

export class FetchHttpClient implements HttpClient {
  fetch(url: string, init: RequestInit): Promise<Response> {
    return fetch(url, init);
  }
}
