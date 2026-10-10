# VS-01 — Action contracts / gaps / invariants
Date 2026-10-10. Logical review, not DDL proof or deployed implementation.
Predecessors: `docs/business/BUSINESS-ATLAS-V1-2026-10-10.md`; BUSINESS-01, -02, -03.1..03.5 in prior chat. Existing baseline: canonical V0.3 160 tables, not feature rollout.

## 1. Product experiment boundary
One ABC team and An/Bình/Hùng; one house 75 m2 received on Zalo, first report VND 4.2bn; second S1 report VND 4.1bn; optional other source S2 VND 4.3bn; prepare content without invented proof; if permission exists, human manually posts on Facebook; Minh enquires by personal Zalo, doesn't join platform; An records minimal request and next action; optionally hands to Bình; visit agreed/cancelled/held; report qualified outcomes. **This is a hypothetical example, not real data.** No marketplace network, no personal Zalo read automation, no mass auto-posting.

## 2. Cross-cutting invariant ledger
- `ID-01` Account authenticates a Person; a Person may hold roles at multiple Orgs; C01 Party exactly one Person or Org still lacks PostgreSQL proof.
- `AUTH-01` org membership alone doesn't grant blanket resource rights; check account, current membership/role, object scope, purpose, policy and high-risk operation at commit.
- `TENANT-01` every org-scoped link/lookup/query/projection/cache/job/media export must enforce same eligible Org, including nullable references. An/ABC cannot dedupe/search XYZ contacts.
- `DECL-01` reported information != source confirmation != ownership/title/publishing authorization.
- `TIME-01` capture time, source statement time, effective time, export time and published time differ; unknown may stay null.
- `TX-01` each declared one-click multi-write action is atomic; failures don't show false success; idempotency key prevents duplicate replay, not intentionally independent reports.
- `CX-01` concurrent changes should not silently overwrite; version check or locking + current authority.
- `P-01` personal contact data minimized, limited by purpose; no unconditional marketing from a single inquiry; redact audit and snapshots.
- `PUB-01` preparing internal draft != exporting != public publication != verified external result.
- `OBS-01` dashboard and metrics only claim recorded or explicitly qualified events, not invisible external activity.

## 3. Contract: CaptureSource
**Actor:** An, authenticated Account in ABC; an independent broker personal workspace is a *different* scope design still open.
**User inputs:** declared house, area 75 m2 with unknown area basis, channel Zalo, optional source note, initial price VND 4.2bn; request Confirm (due time may be unset).
**Server-provided:** account/person, verified org context and current Membership, operation key scope/hash, recorded_at, version.
**Preconditions:** active principal, permission to add in ABC, values valid/positive, source disclosure compatible with purpose; never trust created_by from client.
**Transaction:** create source intake; if initial price given create price report with same source/time provenance; if UI promises Confirm task create it consistently; store idempotency outcome. Return IDs/version plus classification 'reported/unverified'. No Property/Listing/publication.
**Failure:** no membership, revoked access, reusing key with differing payload, network lost after commit, invalid currency, two genuine independent sources. Allow source even before physical address or Property known.
**Open:** independent personal custody, whether confirm remains inline state vs dedicated task.

## 4. Contract: ConfirmOrChangeSourcePrice
**Actor:** authorized An/ABC, permission on source S1; maybe another account reports from S2.
**Inputs:** current version, VND 4.1bn, who was consulted and how, received_at/observed_at (if known), evidence basis, request key.
**Transaction:** compare version/authority; append meaningful price report without rewriting S1 history; update controlled current-version relation or derive latest admissible price; optionally finish actual 'call source' task with separate activity result; audit limited metadata.
**Must not:** rewrite S2 report; replace all `listing_price_terms`; assert title/ownership; automatically notify all CRM Contacts; claim existing social posts updated.
**Race:** 09:00 source says 4.1, 10:00 another says 4.05, 11:00 first statement entered: last database insertion isn't automatically newest real-world information.
**Open:** price term basis: asking/owner-net/tax-included; validation/conflict policy by source, data retention.

## 5. Contract: PrepareShare
**Actor:** user with read/draft permission on source; data scope ABC.
**Read:** versioned price/status/location/detail and allowed media, with field-level restrictions. Generate from known facts only, no 'red book verified', 'owner selling urgently', fabricated address or legal assurance. Unknown remains unknown.
**Write:** none necessary for preview. Return draft with source version and disclosure warnings.
**Open:** tenant media metadata and rights chain, source photos before Property identification; template/copy-only first.
## 6. Contract: ExportShare
**Actor:** user with resource-specific export/publish capacity and media-use rights; rights rechecked at point of export.
**Inputs:** source revision expected, selected fields/media, destination class.
**Preconditions:** authorized for particular external exposure; adequate evidence where required by transaction/legal context; sensitive contacts and images excluded.
**Outcome:** export content, possibly store a minimal attributable snapshot/export receipt if product needs audit. Copy success != posted; no auto-platform permissions.
**Failure:** revision changed, media rights revoked, company member offboarded, unapproved content, false claims.
**Future:** CMS editorial review and authorized integrations separately.

## 7. Contract: CaptureInquiry
**Actor:** An receives Minh's Zalo message; asserted acquisition Facebook; contact channel Zalo; capture is BDSPro UI. Minh has no BDSPro Person or Account.
**Inputs:** message gist 'house still available? weekend viewing', optional allowed contact details, source intake ref, acquisition_basis=customer_reported, permitted next action.
**Preconditions:** current ABC authority and view source, lawfully handled data and purpose.
**Transaction:** create Inquiry event/request and if user asked create a Follow-up with accountable assignee; link Contact if known under same org, never auto-merge unknown persons. Idempotent retry, independent later contact is a distinct event.
**Must not:** create Party, compulsory Contact, compulsory Opportunity pipeline, claim Facebook click or Zalo personal access, grant marketing rights, leak XYZ CRM.
**Open:** table vs smaller Work Item first, retention and consent evidence as legally appropriate.

## 8. Contract: AssignOrCompleteFollowUp
**Truth:** Work Item = unfulfilled/current responsibility, Activity = event already happened.
**Start:** OPEN + assignee within same Org, optional due_at. OVERDUE is derived due_at < now if still OPEN; 'sent reminder' is delivery event not work status.
**Close:** COMPLETED or CANCELED, never both. Store outcome required by type; optionally append activity. Completion is not automatic proof of sale or customer satisfaction.
**Transfer:** Hùng → Bình retains An as intake originator, explicit current membership permission, version guarded.
**Race:** Hùng transfers while An clicks completed on stale screen → one transition is rejected/reconciled; genuine call before transfer can still have an Activity without unauthorized task closure.
**Failure:** revoked membership, wrong-organization assignee, duplicate retry, notification sent twice, no due date but promised reminder.
**Open:** personal tasks, same-tenant composite FKs and person/account event model.

## 9. Contract: AgreeAppointmentAndRecordOutcome
**Distinctions:** proposed time ≠ mutually agreed; calendar scheduled ≠ visit occurred; canceled ≠ unsuccessful client; attended ≠ Deal.
**Actors:** An and external Minh; neither external customer Account nor platform Person required.
**Write:** appointment proposal, confirmed event by reliable evidence, cancel/reschedule, check-in/visit outcome with an accountable recorder; optional new follow-up.
**Failure:** An and Minh reschedule concurrently, source sold before viewing, client rejects further marketing, appointment bridge currently `appointment_participants.person_id` mandatory.
**Open:** model support CRM Contact/external participant, explicit status and work context; build after pilot uses appointment outcome.

## 10. Contract: MatchAndReport
**Read model:** scoped sources + verified freshness + allowed purpose; deterministic filters (transaction type, locality, budget, area, customer exclusions); show candidate and reason. No blanket cross-tenant query followed by UI filtering.
**Tracking:** distinguish reported inquiry, qualified inquiry, follow-up action, proposed visit, confirmed visit, actual attended, Deal and cleared commercial outcome. Numerator/denominator and external capture coverage.
**Report:** user gets Today dashboard and team manager sees actionable backlog; report derived from recorded facts; do not treat invisible Zalo work as zero performance.
**Economics:** compare against baseline Facebook/Zalo/Sheets with matched lead/stock volumes and ads spend; don't attribute all completed deals to CRM.

## 11. Physical gap ledger (not permission to bulk implement)
| Pressure | Current V0.3 | Proposal / candidate | Status |
|---|---|---|---|
| Unverified incoming stock | properties/listings don't mean this | `source_intakes` | REVIEW CANDIDATE |
| Prices by source and observation time | listing_price_terms scoped to Listing | `source_price_reports` | REVIEW CANDIDATE |
| External inquiry, source attribution | crm_contacts/opportunities/activities overloaded if used alone | `crm_inquiries` | REVIEW CANDIDATE |
| Responsibility with deadline/status | crm_activities is event only | `crm_followup_tasks` | REVIEW CANDIDATE |
| Changing buying/rental demand vs forecast value | crm_opportunities.expected_amount is forecast | `crm_property_demands` | CONDITIONAL |
| Media before verified Property | property_media requires property | scoped intake media/permission contract | OPEN |
| External sharing receipt | listing_publications refers to Listing | render-only then optional export receipt | DEFER |
| Appointment without Account/Person | appointment_participants requires Person | CRM/external attendee strategy | CONDITIONAL |
| CMS/news/SEO/marketing campaign governance | no first-class editorial lifecycle | separate CMS/marketing candidate family | DEFER; KNOWN GAP |
| Same-tenant integrity of existing CRM bridges | independent FKs | composite keys or guarded write + tests | HIGH PRIORITY |
| One Party has exactly one subtype | Party FK isn't enough | C01 lab transaction proof | HIGH PRIORITY |
| Enterprise representative legitimacy | organization claim/representation candidates | real evidence and review queue | CONDITIONAL |

**Implementation default:** VS-01 TEAM-only scope explicitly, not a false general personal/org ownership model. Personal Workspace remains an actual different product decision; do not choose polymorphic owner string just for reuse.

## 12. Test case ledger
T01 valid capture and refresh; T02 retry same key identical response; T03 same key changed payload rejects; T04 two independent sources same house not auto-merged; T05 4.2→4.1 source history preserved; T06 source S2 4.3 not overwritten; T07 late event-time report doesn't become newest by insert time; T08 user ABC tries XYZ source or contact fails; T09 revoked ABC Member cannot create/finish/export; T10 two edits same expected version → one recognized conflict; T11 unauthorized photo excluded from export; T12 export not reported as verified external post; T13 Minh without Account creates followup; T14 same Minh cross-tenant no existence leak; T15 appointment cancel doesn't cancel CRM identity; T16 manager report excludes unrecorded external activity; T17 no timed reminder for null due_at; T18 concurrent reassignment/completion enforces allowed serial outcome; T19 no new public Listing from intake; T20 data erasure/retention policy tested with legitimate business history/legal exceptions.

## 13. Acceptance criteria and counterexamples
A feature is accepted only with: a real job, who benefits/pays, explicit one-row invariant, actor/purpose authorization, pre/postconditions, race/retry/tenant/PII failure tests, metric and baseline, operator burden, reversible/expand migration plan. Unproven market tests retain 'hypothesis'. All SQL patterns remain illustrative until executed and verified against actual `dev` migrations, code and PostgreSQL.

## 14. Next
See `BUSINESS-TO-CORE-IMPLEMENTATION.md` for gate-by-gate code progression; do not use the above as blanket approval to migrate 160 tables. `docs/continuity/checkpoints/2026-10-10-business-to-core/README.md` is the resume anchor.
