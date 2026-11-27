export class ApiError extends Error {
  constructor(status, code, requestId) {
    super(code);
    this.status = status;
    this.code = code;
    this.requestId = requestId;
  }
}

const knownCodes = new Set([
  'invalid_credentials', 'authentication_required', 'token_expired',
  'validation_failed', 'invalid_verification_code', 'invalid_proof',
  'permission_denied', 'reauthentication_required', 'credential_conflict',
  'username_unavailable', 'rate_limited', 'service_unavailable',
  'resource_not_found', 'internal_error', 'delivery_channel_unavailable',
]);

export async function api(path, { method = 'GET', body, token, signal } = {}) {
  if (!path.startsWith('/') || path.startsWith('//')) throw new Error('Invalid API path');
  const headers = { Accept: 'application/json' };
  if (method !== 'GET') headers['X-BDS-Request'] = '1';
  if (body !== undefined) headers['Content-Type'] = 'application/json';
  if (token) headers.Authorization = 'Bearer ' + token;
  const response = await fetch('/api/v1' + path, {
    method, headers, credentials: 'include', cache: 'no-store',
    body: body === undefined ? undefined : JSON.stringify(body),
    signal: signal || AbortSignal.timeout(15000),
  });
  if (response.status === 204) return null;
  const data = await response.json().catch(() => null);
  if (!response.ok) {
    const raw = data?.code ?? data?.error?.code;
    const code = knownCodes.has(raw) ? raw :
      ({401:'authentication_required',403:'permission_denied',404:'endpoint_not_found',409:'credential_conflict',429:'rate_limited',503:'service_unavailable'}[response.status] || 'request_failed');
    const id = response.headers.get('X-Request-ID') ?? data?.request_id ?? data?.error?.request_id;
    throw new ApiError(response.status, code, typeof id === 'string' && /^[a-zA-Z0-9_-]{1,100}$/.test(id) ? id : undefined);
  }
  if (data === null) throw new ApiError(response.status, 'invalid_response');
  return data;
}

export function assertAuthenticated(data) {
  if (data?.status !== 'authenticated' || typeof data.access_token !== 'string' ||
      !data.access_token || typeof data.user?.id !== 'string' || !data.user.id) {
    throw new ApiError(200, 'invalid_response');
  }
  return { token: data.access_token, user: data.user };
}

export const messages = {
  delivery_channel_unavailable: 'This delivery channel is unavailable. Choose another channel.',
  endpoint_not_found: 'This endpoint is not available in the current backend. Check the IAM milestone.',
  invalid_credentials: 'Invalid email, username or password.',
  authentication_required: 'Please sign in again.',
  token_expired: 'Your session has expired.',
  validation_failed: 'Please check the form fields.',
  invalid_verification_code: 'The verification code is invalid or expired.',
  invalid_proof: 'Verification expired. Start again.',
  permission_denied: 'You do not have permission for this action.',
  reauthentication_required: 'Please verify your identity again.',
  username_unavailable: 'Choose another username.',
  credential_conflict: 'This credential cannot be linked.',
  rate_limited: 'Too many requests. Please wait and retry.',
  service_unavailable: 'The service is temporarily unavailable.',
  invalid_response: 'The backend response does not match the IAM contract.',
  request_failed: 'The request failed. Please try again.',
  internal_error: 'The backend could not complete this request.',
};
