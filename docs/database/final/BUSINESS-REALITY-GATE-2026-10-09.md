# BDSPro — Business Reality Gate: Facebook/Zalo-first real-estate workflows

**Date:** 2026-10-09. **Status:** BUSINESS HYPOTHESIS REVIEW, NOT VALIDATED CUSTOMER DEMAND.  
**Applies to:** 160-table canonical V0.3 candidate; **NO DBML or production schema changes** authorized by this note.  
**Question:** Does BDSPro create value **before** it has a marketplace audience, qualified supply or its own community?

## 1. Key counterexample to earlier architecture assumptions

A broker posts manually to Facebook Groups, Zalo personal/group chats and property portals, calls customers, visits sites and reports activity in Sheets/Zalo. The prospecting audience already exists elsewhere. BDSPro is initially an extra screen with **zero incremental native demand**. A new Group, community badge, reputation score, managed identity, GIS engine, enterprise hierarchy or complicated Deal workflow does not automatically improve this day.

**Architectural correction:** The **north-star ecosystem remains an ambition**, but the first product must be an independent utility that saves time, prevents loss of valid leads, keeps property stock accurate, helps appropriate client follow-up or proves better conversions **without requiring platform network effects**. This is a new prioritization, not automatic deletion of tables.

## 2. Honest value proposition / adoption equation

The user's weekly benefit must clearly exceed:
- time entering and correcting CRM records;
- time switching away from Facebook/Zalo/phone;
- fear of company/platform appropriating personal customer relationships;
- training, payments and integration limitations;
- manager reporting burden and unwanted surveillance.

Break-even (illustrative, never assumed factual):
`net value = incremental gross profit attributable to tool + value of hours actually saved - cash subscription - data-entry/time costs - onboarding/integration cost - privacy/operational risk exposure`.
Do NOT attribute all deals to CRM or claim time saved equals new cash.

## 3. Job-to-be-done observations, NOT yet validated survey

| Job | Existing approach | Plausible friction | First cheap experiment | Stop signal |
| --- | --- | --- | --- | --- |
| Source inventory | Zalo group, Facebook, Sheets, notes | stale price/availability, duplicate listings | central property card + stock update + link share | users prefer existing shared sheet and won't edit card |
| Promotion | manually posting in Facebook Groups and Zalo channels | repetitive formatting, lost provenance | copy-ready caption, image pack, short trackable link, manual status log | saves no measurable time or creates extra data entry |
| Lead intake | messages, calls, personal Zalo | missed handoff, no callback, duplicate prospects | one-tap lead/interest, source, owner, next-follow-up, reminders | staff refuse to enter their own leads |
| Team coordination | Zalo group, Excel, voice | two agents call same prospect; orphaned inquiries | lightweight lead assignment, visibility, conflict handling | team does not have enough shared leads to justify management |
| Reporting | spreadsheet or daily Zalo status | manual end-of-day copy, unreliable manager oversight | automatic report derived from already-used workflow | app increases reporting duties and surveillance friction |
| Demand matching | verbal inquiries and agent memory | property-client match missed | saved demand criteria vs actual available stock | insufficient high-quality structured stock and consented demand |
| Compliance/trust | personal statements, contract documents | bad property data and impersonation | selective source evidence and clear verification limitations | verification cost exceeds risk reduction in initial segment |

**First candidate wedge:** lightweight **Property Stock + Lead Follow-up + Share Kit** for a small, demonstrably lead-congested broker team. Not yet assumed to be the winning wedge; compare against standalone lightweight lead follow-up and stock-update-only alternatives.

## 4. Channel reality / official constraints

- **Facebook Groups**: Meta deprecated the public Facebook Groups API and `publish_to_groups` permission as of **2024-04-22**. Do not promise third-party mass posting to arbitrary groups or unofficial scraping/bots without authorization. Default product can prepare assets/reminders for manual use. Evidence: https://help.zapier.com/hc/en-us/articles/23970212345357-App-update-Facebook-Groups-app-removal and https://support.hookle.net/hc/en-us/articles/13668371625500-Publishing-to-Facebook-Groups-is-no-longer-supported
- **Zalo OA vs personal Zalo**: Official Account API and webhooks enable business-owned OA interactions with required application permissions/conditions; this does **not** imply API access to employees' personal chats/groups. Official sources: https://oa.zalo.me/home/documents/guides/Khoi-tao-ung-dung-va-cap-quyen_117071366476220195 and https://oa.zalo.me/home/documents/vie/guides/tong-quan-cac-loai-tin-nhan-tren-zalo-official-account-_3651713298729094511
- **Portals** already have demand; BDSPro must not project own reach from a competitor's traffic.
- **NAR 2026 report** (US only, not direct Vietnam market proof): 81% said saving time was a tech-adoption goal, 71% client experience; 63% say learning curve is a challenge, 59% cost. Source: https://www.nar.realtor/newsroom/realtors-adopt-technology-to-save-time-and-improve-the-client-experience-nar-report-finds
- **NAR 2025** (US only): 39% named social media highest source of quality leads and 23% CRM: https://www.nar.realtor/news/real-estate-news/technology/the-top-tech-tools-giving-real-estate-agents-a-high-tech-edge
- **Vietnam competition**: Batdongsan.com.vn already supplies listings, demand, verified-listing signals and distribution, not directly comparable to new BDSPro cold start: https://batdongsan.com.vn/Landingpage/gioi-thieu/ and https://doanhnghiep.batdongsan.com.vn/tinxacthuc

## 5. Different segments; different payers

| Segment | Demand for another tool | Candidate payer / evidence threshold |
| --- | --- | --- |
| Independent broker with a few active clients | LOW unless one action is dramatically easier | Only if saves ongoing user time and staff will use it weekly |
| 3–15 person broker team sharing active stock/leads | MEDIUM hypothesis | Manager/owner pays only for fewer missed leads, faster handoff and low entry overhead |
| Agency with many sales and paid lead acquisition | Potentially HIGH | Budget holder needs quantified conversion/response/attribution lift over current CRM/tools |
| Developer distribution group/project channel | CONDITIONAL | Verify controls on pricing/stock/authorized partners and current portal/campaign workflow |
| Consumers/homebuyers | No cold-start network value alone | Meaningful unique inventory or trustworthy data; cannot promise national coverage |

A manager wants structured reporting; an agent wants fewer duplicative tasks, more relevant customers and protection of personal relationships. If the agent has no incentive to enter information, the manager's dashboard is empty or fabricated. Design **both sides' incentives**, not only manager authority.

## 6. Concrete minimum pilot / falsification

**Observation phase:** interview/observe ~5 teams with different sizes and ~15–25 agents (targets for study, not validated sample). Record one week of actual daily workflows with consent. Count sources, raw inquiries, missed callbacks, duplicate client contact, property price changes, time spent on post prep, report time, staff willingness to record. Look at actual artifacts, not hypothetical feature requests.

**Low-code/no-code prototype** (not necessarily a custom database first): one property card, one share kit, one prospect card with minimum required fields, one next action, a short manager report. Manual channel posting remains. No automated scraping, no full CRM or complex enterprise IAM to start.

**Pilot:** 2–4 weeks with participating team; compare same team's baseline, seasonality and advertising budget. Candidate acceptance criteria (to agree with participants): meaningful daily or weekly retention for performing staff; fewer unreplied valid inquiries, shorter response latency, less report time, lower stale inventory and plausible willingness to pay. Zero cross-tenant privacy defects in tested cases. Do not fake precise numeric KPI thresholds without baseline.

**Business falsifier:** if users won't maintain data without a supervisor forcing them, lead import is more costly than value, team can't identify a problem worth paying to remove, or no meaningful uplift survives a pilot, **STOP/SIMPLIFY**. Choose another entry wedge rather than expanding architecture.

## 7. Effects on canonical Core V0.3 (160 candidates)

| Class | Existing table families | New priority |
| --- | --- | --- |
| **Evaluate early only if chosen pilot needs** | `accounts`, `persons`, minimal organization context/membership, `properties`, `property_locations`, `listings`, `listing_properties`, `crm_contacts`, `crm_opportunities`, activity/follow-up | Decide simplest one-row facts and tenant access; resist rebuilding every identity subtype before product proof |
| **Potential first critical gaps** | lead/source attribution, actual inquiry intake, follow-up ownership, property stock freshness, consent, reference/link to off-platform posting | Do not invent new generic Social/Channel engine; clarify minimal event/state representation per use case |
| **Conditional as team's scale demands** | `organization_teams`, `organization_team_assignments`, `organization_roles`, `subscriptions`, `usage_events` | Team may be a simple assignment first, not hierarchical department/managed accounts |
| **Defer until evidence and audience** | `group_organization_participations`, `communities`, social graph, most Deal/investment/commission/auctions, full enterprise managed IAM | Preserve candidate semantics, DO NOT implement because DBML includes tables |
| **Defer expensive source pipelines** | `planning_*`, `cadastral_parcels`, map tile/report engines | Focus on proven legal access + a small geography/use case only if customers pay for info |
| **Defer growth scores** | `reputation_policy_versions`, `reputation_assessments`, `reputation_benefit_grants` | A scoring system has no meaningful verified evidence or platform audience at cold start |
| **Cross-cutting essentials** | access isolation, audit/idempotency only when failure risks demand | Small implementation does NOT waive privacy, honest data sources or safe auth |

**Nothing here changes the count or content of the 160-table V0.3 DBML.** This document revises **business priority and investment authority**. Table count does not equal backlog commitment.

## 8. Cost/benefit discipline: the investment ladder

1. Manual workflow observation and whiteboard.
2. Google Sheets/Notion/prototype tests where permitted by privacy policy.
3. Very small Go/PostgreSQL slice if actual work has regular repetition, concurrent edits, ownership or confidential tenant data.
4. More schemas/authorization policies only when recurring real use demands them.
5. Integration only with approved channel APIs and sustainable maintenance budgets.
6. Network/community/reputation/search only after genuine supply/demand and sufficient evidence.
7. National data, GIS, eKYC, regulated settlement only with licensed sources/contracts, security and legal budget.

**No premature pivot to "just CRM"**: that too is a crowded product category. The goal is an **earned wedge** where domain-specific property+social workflow beats existing habits/tools and competes on measurable ROI, UX and trust.

## 9. Revised next milestone

**BUSINESS-00: Reality and willingness-to-pay study** should happen **before authorizing production implementation** of ORG-01/ORG-NET-01. Core conceptual review can continue; C01/DB-21 lab remains valid *learning*, not validation of a paying customer.

Next user/research meeting: replay one real team's workflow from source inventory → external post → inquiry → qualification → follow-up → customer visit → deal/report, marking exact time, responsibility, failure, sensitivity and value. Only then choose the first earning invariant and minimal tables.

**Decision:** Keep trust/data/community national ambition as a north star, but **move go-to-market and operating reality ahead of broad architectural completion**. Label claims as field-observation hypotheses until observed, paid pilot outcomes confirm. Stop investing in capabilities whose users/value are not demonstrated.
