import React, { useEffect, useState } from 'react';
import { createRoot } from 'react-dom/client';
import { api, assertAuthenticated, messages } from './api.js';
import './styles.css';

const initialScreen = location.pathname === '/auth/google/return' ? 'googleReturn' : 'register';
const titles = {
  register: 'Create BDSPro account', login: 'Sign in to BDSPro',
  verifyEmail: 'Verify your email', phone: 'Continue with phone',
  verifyPhone: 'Enter your verification code', complete: 'Complete your account',
  forgot: 'Forgot your password?', reset: 'Reset your password',
  googleReturn: 'Continue with Google', account: 'Your account',
};

function Password({ value, onChange, autoComplete = 'current-password', label = 'Password', minLength }) {
  const [visible, setVisible] = useState(false);
  return <label className="field">
    <span className="label-row">{label}<button type="button" className="text-button" onClick={() => setVisible(!visible)} aria-pressed={visible}>{visible ? 'Hide' : 'Show'}</button></span>
    <input required minLength={minLength} type={visible ? 'text' : 'password'} autoComplete={autoComplete} value={value} onChange={e => onChange(e.target.value)} />
  </label>;
}

function App() {
  const [screen, setScreen] = useState(initialScreen);
  const [email, setEmail] = useState('');
  const [username, setUsername] = useState('');
  const [identifier, setIdentifier] = useState('');
  const [password, setPassword] = useState('');
  const [phone, setPhone] = useState('');
  const [country, setCountry] = useState('VN');
  const [code, setCode] = useState('');
  const [remember, setRemember] = useState(true);
  const [marketing, setMarketing] = useState(false);
  const [terms, setTerms] = useState(false);
  const [challenge, setChallenge] = useState(null);
  const [proof, setProof] = useState(null);
  const [method, setMethod] = useState('email');
  const [session, setSession] = useState(null);
  const [resetToken, setResetToken] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');
  const [health, setHealth] = useState('Checking API…');
  const [now, setNow] = useState(Date.now());

  useEffect(() => {
    const timer = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(timer);
  }, []);
  useEffect(() => { checkHealth(); }, []);

  async function checkHealth() {
    setHealth('Checking API…');
    try {
      const data = await api('/health');
      setHealth(data?.status === 'ok' ? 'Go API connected' : 'Unexpected health response');
    } catch {
      setHealth('Go API unavailable');
    }
  }

  function go(next) {
    setScreen(next); setError(''); setNotice(''); setPassword(''); setCode('');
    setProof(null); setChallenge(null); setResetToken('');
  }

  async function run(fn) {
    if (busy) return;
    setBusy(true); setError(''); setNotice('');
    try { await fn(); } catch (err) {
      const message = messages[err.code] || 'Cannot complete the request. Check the backend connection.';
      setError(message + (err.requestId ? ' Reference: ' + err.requestId : ''));
    } finally { setBusy(false); }
  }

  function accept(data, kind) {
    if (data?.status === 'authenticated') {
      const result = assertAuthenticated(data);
      setSession(result); setProof(null); setChallenge(null); setPassword(''); setCode('');
      setScreen('account'); setNotice('Signed in using the real backend.');
    } else if (data?.status === 'registration_required' && typeof data.registration_token === 'string' && data.registration_token) {
      setProof(data.registration_token); setMethod(kind); setPassword(''); setCode('');
      setScreen('complete');
    } else if (data?.status === 'sign_in_required') {
      go('login'); setNotice('Sign in to your existing account or use account recovery.');
    } else {
      throw new Error('Unexpected authentication state');
    }
  }

  function acceptChallenge(data, next) {
    if (data?.status !== 'verification_required' || typeof data.challenge_id !== 'string') throw new Error('Unexpected challenge response');
    setChallenge({ id: data.challenge_id, resendAt: Date.now() + (Number(data.resend_after) || 0) * 1000 });
    setScreen(next); setPassword(''); setCode('');
  }

  const post = (path, body, token) => api(path, { method: 'POST', body, token });
  const submit = e => {
    e.preventDefault();
    run(async () => {
      if (screen === 'register') {
        acceptChallenge(await post('/auth/register/start', {email}), 'verifyEmail');
      } else if (screen === 'login') {
        accept(await post('/auth/login', {identifier,password,stay_signed_in:remember}), 'email');
      } else if (screen === 'phone') {
        acceptChallenge(await post('/auth/phone/start', {phone,country_code:country}), 'verifyPhone');
      } else if (screen === 'verifyEmail' || screen === 'verifyPhone') {
        const kind = screen === 'verifyEmail' ? 'email' : 'phone';
        accept(await post(kind === 'email' ? '/auth/register/verify' : '/auth/phone/verify', {challenge_id:challenge.id,code,...(kind === 'phone' ? {stay_signed_in:remember} : {})}), kind);
      } else if (screen === 'complete') {
        if (!terms || !proof) throw new Error('Missing registration proof or consent');
        const payload = {registration_token:proof,username,stay_signed_in:remember,terms_version:'v1',marketing_opt_in:marketing};
        if (method === 'email') payload.password = password;
        accept(await post('/auth/' + (method === 'email' ? 'register' : method) + '/complete',payload), method);
      } else if (screen === 'forgot') {
        await post('/auth/password/reset/request',{email});
        setScreen('reset'); setNotice('If eligible, recovery instructions will be sent.');
      } else if (screen === 'reset') {
        await post('/auth/password/reset/confirm',{reset_token:resetToken,new_password:password});
        go('login'); setNotice('Password reset. Sign in again.');
      }
    });
  };
  const resendSeconds = challenge ? Math.max(0, Math.ceil((challenge.resendAt - now)/1000)) : 0;
  const resend = () => run(async () => acceptChallenge(await post(screen === 'verifyEmail' ? '/auth/register/resend' : '/auth/phone/resend',{challenge_id:challenge.id}),screen));
  const google = () => { location.assign('/api/v1/auth/google/start?stay_signed_in=' + remember); };
  const refresh = () => run(async () => accept(await post('/auth/refresh',{}),'email'));
  const logout = () => run(async () => { await post('/auth/logout',{}); setSession(null); go('login'); });
  const showAlternatives = screen === 'register' || screen === 'login';
  const showPreferences = screen === 'register' || screen === 'complete';
  const showRemember = showPreferences || screen === 'login' || screen === 'verifyPhone';
  const submitLabel = {register:'Create Free Account',login:'Sign in',phone:'Send verification code',verifyEmail:'Verify email',verifyPhone:'Verify phone',complete:'Create account',forgot:'Send recovery instructions',reset:'Reset password'}[screen];

  return <main>
    <section className="shell" aria-labelledby="title">
      <div className="brand">BDS<span>Pro</span></div>
      <h1 id="title">{titles[screen]}</h1>
      <div className="status"><span>{health}</span><button type="button" className="text-button" onClick={checkHealth}>Check again</button></div>
      <p className="lab-note">IAM development client. Real API calls only; unimplemented routes will fail.</p>
      {error && <p role="alert" className="error">{error}</p>}
      {notice && <p role="status" className="notice">{notice}</p>}
      {screen === 'account' ? <div className="account">
        <p>Signed in as <strong>{session?.user?.username || session?.user?.id}</strong></p>
        <dl>
          <dt>Email</dt><dd>{session?.user?.email || 'Not linked'}</dd>
          <dt>Phone</dt><dd>{session?.user?.phone || 'Not linked'}</dd>
          <dt>Password</dt><dd>{session?.user?.credentials?.password_set ? 'Configured' : 'Not configured'}</dd>
          <dt>Google</dt><dd>{session?.user?.credentials?.google_linked ? 'Connected' : 'Not connected'}</dd>
        </dl>
        <button className="secondary" disabled={busy} onClick={() => run(async () => {
          const user = await api('/me',{token:session.token});
          if (user?.id !== session.user.id) throw new Error('Unexpected user identity');
          setSession(old => ({...old,user}));
        })}>Reload current user</button>
        <button className="secondary" disabled={busy} onClick={refresh}>Refresh session</button>
        <button className="primary" disabled={busy} onClick={logout}>Sign out</button>
        <p className="lab-note">Full profile, credential, session and organization screens are required by FE-007–009; not yet implemented in this starter.</p>
      </div> : screen === 'googleReturn' ? <>
        <button className="primary" disabled={busy} onClick={() => run(async () => accept(await post('/auth/google/result',{}),'google'))}>Complete Google sign-in</button>
        <button className="text-button" onClick={() => { history.replaceState(null,'','/'); go('login'); }}>Back to sign in</button>
      </> : <form onSubmit={submit}>
        {(screen === 'register' || screen === 'forgot') && <label className="field">Work email<input required type="email" autoComplete="email" value={email} onChange={e=>setEmail(e.target.value)} /></label>}
        {screen === 'login' && <label className="field">Email or username<input required autoComplete="username" value={identifier} onChange={e=>setIdentifier(e.target.value)} /></label>}
        {showPreferences && <label className="field">Username<input required minLength={3} maxLength={30} pattern="[A-Za-z0-9_-]+" autoComplete="username" value={username} onChange={e=>setUsername(e.target.value)} /></label>}
        {(screen === 'register' || screen === 'login' || screen === 'reset' || (screen === 'complete' && method === 'email')) && <Password value={password} onChange={setPassword} minLength={screen === 'login' ? undefined : 15} autoComplete={screen === 'login' ? 'current-password' : 'new-password'} />}
        {screen === 'register' && <p className="hint">Verify email first. You will confirm your password on the next step.</p>}
        {screen === 'phone' && <>
          <label className="field">Country code<input required maxLength={2} value={country} onChange={e=>setCountry(e.target.value.toUpperCase())} /></label>
          <label className="field">Phone number<input required type="tel" autoComplete="tel" value={phone} onChange={e=>setPhone(e.target.value)} /></label>
        </>}
        {(screen === 'verifyEmail' || screen === 'verifyPhone') && <>
          <label className="field">Verification code<input required inputMode="numeric" autoComplete="one-time-code" value={code} onChange={e=>setCode(e.target.value)} /></label>
          <button type="button" className="text-button" disabled={busy || resendSeconds > 0} onClick={resend}>{resendSeconds ? 'Resend in ' + resendSeconds + 's' : 'Resend code'}</button>
        </>}
        {screen === 'reset' && <label className="field">Recovery token<input required type="password" autoComplete="off" value={resetToken} onChange={e=>setResetToken(e.target.value)} /></label>}
        {showPreferences && <label className="check"><input type="checkbox" checked={marketing} onChange={e=>setMarketing(e.target.checked)} />Receive product updates, news, and other marketing communications</label>}
        {showRemember && <label className="check"><input type="checkbox" checked={remember} onChange={e=>setRemember(e.target.checked)} />Stay signed in</label>}
        {showPreferences && <>
          <label className="check"><input required type="checkbox" checked={terms} onChange={e=>setTerms(e.target.checked)} />I accept the development-only terms and privacy notice (v1).</label>
          <p className="hint">Test consent only. Approved legal terms and privacy URLs are a production gate.</p>
        </>}
        <button type="submit" className="primary" disabled={busy}>{busy ? 'Please wait…' : submitLabel}</button>
        {screen === 'login' && <button type="button" className="text-button" onClick={()=>go('forgot')}>Forgot password?</button>}
      </form>}
      {showAlternatives && <>
        <div className="separator"><span>or</span></div>
        <button type="button" className="secondary provider" disabled={busy} onClick={google}><span className="google-mark" aria-hidden="true">G</span>Continue with Google</button>
        <button type="button" className="secondary" disabled={busy} onClick={()=>go('phone')}>Continue with phone number</button>
        <p className="switch">{screen === 'register' ? 'Already have an account?' : 'New to BDSPro?'} <button className="text-button" disabled={busy} onClick={()=>go(screen === 'register' ? 'login' : 'register')}>{screen === 'register' ? 'Sign in' : 'Create account'}</button></p>
        <button className="text-button" disabled={busy} onClick={refresh}>Restore existing session</button>
      </>}
      {!showAlternatives && !['account','googleReturn'].includes(screen) && <button type="button" className="text-button back" disabled={busy} onClick={()=>go('login')}>Back to sign in</button>}
    </section>
  </main>;
}

createRoot(document.getElementById('root')).render(<App />);
