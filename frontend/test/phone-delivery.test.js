import test from 'node:test';
import assert from 'node:assert/strict';
import {api, ApiError} from '../src/api.js';

test('phone start and resend preserve selected channel and browser credentials', async t => {
  const calls = [];
  t.mock.method(globalThis, 'fetch', async (url, options) => {
    calls.push({url, options});
    return Response.json({status:'verification_required', challenge_id:'test', delivery_channel:'telegram'});
  });
  await api('/auth/phone/start', {method:'POST', body:{phone:'0912345678', country_code:'VN', channel:'telegram'}});
  await api('/auth/phone/resend', {method:'POST', body:{challenge_id:'test', channel:'zalo'}});
  assert.equal(JSON.parse(calls[0].options.body).channel, 'telegram');
  assert.equal(JSON.parse(calls[1].options.body).channel, 'zalo');
  for (const call of calls) {
    assert.equal(call.options.credentials, 'include');
    assert.equal(call.options.headers['X-BDS-Request'], '1');
  }
});

test('flat backend errors preserve code and request ID without exposing provider details', async t => {
  t.mock.method(globalThis, 'fetch', async () => Response.json({
    code:'delivery_channel_unavailable', error:'choose an available verification channel'
  }, {status:400, headers:{'X-Request-ID':'server-reference-123'}}));
  await assert.rejects(api('/auth/phone/start', {method:'POST', body:{channel:'zalo'}}),
    e => e instanceof ApiError && e.code === 'delivery_channel_unavailable' && e.requestId === 'server-reference-123');
});
