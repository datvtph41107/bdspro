# BDSPro IAM React starter

Development acceptance client, inspired by the supplied Postman form. No personal screenshot data is committed. **Not a completed IAM product.** The current Go baseline has only health and four protected Team endpoints, so login/register/phone/Google/recovery requests report real backend errors until IDE agent implements them. The UI includes initial forms and basic current-user/session calls; final settings, linking, org/team, automatic refresh coordination, browser tests and all backend implementations are required by docs/iam. No mocked authentication success.

Node 22.12+ (maintained LTS recommended). React 19.3.1 and Vite 8.3.8 selected from npm listing at authoring; check compatibility and audit before release. npm lockfile and build are not verified in the authoring environment (registry DNS unavailable). First local setup:

```sh
# repo root, isolated dev database and Redis configured
make config-local
# review custom/conf/app.ini before migration
make migrateup
make server
```

Second terminal:
```sh
cd frontend
npm install
npm test
npm run dev
```

Open http://localhost:5173. Check Go API connected (GET /api/v1/health). Vite proxy target defaults to http://127.0.0.1:8080; set BDS_API_TARGET in shell if necessary. Do not store secrets in frontend env. Commit package-lock.json after actual install; subsequent installs npm ci. Run npm run build and browser acceptance after real backend implementation. The active browser-sensitive mutation routes use the X-BDS-Request: 1 header, configured Origin and Sec-Fetch-Site guard. Session and registration/Google browser bindings remain separate. The backend disposable HTTP probe checks guard admission; full browser/deployment acceptance is still required. Do not use this starter in production. Replace dev terms with approved legal documents before production.

Read ../docs/iam/FRONTEND_OPERATIONS.md and ARCHITECTURE_API.md for full implementation and real E2E requirements.
