import test from 'node:test';
import assert from 'node:assert/strict';
import { api, assertAuthenticated, ApiError } from '../src/api.js';

test('missing endpoint is an error, not fake authentication', async t => {
  t.mock.method(globalThis,'fetch',async () => new Response('not found',{status:404}));
  await assert.rejects(api('/auth/login',{method:'POST',body:{identifier:'sample',password:'not-a-real-secret'}}),e=>e instanceof ApiError && e.status===404);
});

test('API health uses real relative path', async t => {
  t.mock.method(globalThis,'fetch',async (url,opts) => {
    assert.equal(url,'/api/v1/health');
    assert.equal(opts.credentials,'include');
    return Response.json({status:'ok'});
  });
  assert.deepEqual(await api('/health'),{status:'ok'});
});

test('reject malformed authenticated result', () => {
  assert.throws(()=>assertAuthenticated({status:'authenticated',user:{id:'1'}}));
  assert.deepEqual(assertAuthenticated({status:'authenticated',access_token:'test-only',user:{id:'1'}}),{token:'test-only',user:{id:'1'}});
});
