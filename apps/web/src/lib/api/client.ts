export const authenticationRequiredEvent = 'flex:authentication-required'

const publicAuthenticationPaths = new Set(['/api/auth/status', '/api/auth/setup', '/api/auth/login', '/api/auth/logout'])

export async function apiFetch(input: RequestInfo | URL, init?: RequestInit): Promise<Response> {
  const response = await fetch(input, init)
  const path = typeof input === 'string' ? (input.split('?')[0] ?? input) : input instanceof URL ? input.pathname : new URL(input.url, window.location.origin).pathname
  if (response.status === 401 && !publicAuthenticationPaths.has(path) && typeof window !== 'undefined') {
    window.dispatchEvent(new CustomEvent(authenticationRequiredEvent))
  }
  return response
}
