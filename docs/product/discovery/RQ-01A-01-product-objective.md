# RQ-01A.01 — Product objective: developer response and challenge
2026-11-25. OPEN / provisional. This is the first of five sequential discovery topics, not an accepted business requirement. Do not advance to customer segment discovery before the owner reviews this checkpoint. Product authority and direct customer validation remain outstanding.

## Reported intent and developer choices
- 01.1 F: business model UNKNOWN.
- 01.2 F: broad ecosystem value; not a testable primary outcome.
- 01.3 E: private workspace + public platform (developer selection, not access policy).
- 01.4 D: long-term ecosystem vision. Backend owner reports manager wants individual/business accounts, B2B/SaaS, listing, SEO/marketing, chat/groups/community, CRM, deals, service/listing packages and AI. Manager's reported intent is not customer demand or authorization to build all capabilities.
- 01.5 E: developer proposes planning/property-information lookup first. Developer reports current work includes listing, user and property management; priority mismatch to resolve, not a confirmed roadmap.
- 01.6 F: actual advantage UNKNOWN, including data rights/coverage and operating capacity.
- 01.7 A/D/E/F/C/B: all six outcome signals selected. Distinguish first-workflow acceptance from later adoption/commercial/marketplace outcomes; no targets yet.
- 01.8 A: manager verbal direction reported by developer; no independent confirmation or written product approval.

## Evidence and limits
Legacy snapshot 0b16ec4b46b1c8813e93a1867f999f3aebd9c97a:
- [TQD parcel controller L31-50](https://github.com/datvtph41107/bdspro-backend/blob/0b16ec4b46b1c8813e93a1867f999f3aebd9c97a/v4/modules/tqd/controller/parcel_client_controller.go#L31-L50) registers parcel text/map/land number search and invokes usecase.
- [Planning controller L53-66](https://github.com/datvtph41107/bdspro-backend/blob/0b16ec4b46b1c8813e93a1867f999f3aebd9c97a/v4/modules/tqd/controller/planning_client_controller.go#L53-L66) registers project/document/layer routes.
- [Property/listing/deal README](https://github.com/datvtph41107/bdspro-backend/blob/0b16ec4b46b1c8813e93a1867f999f3aebd9c97a/v4/modules/bdspro/README.md#L14-L37) establishes competing legacy capabilities, not delivery status.
- Current dev baseline from SOURCE_BASELINE: no demonstrated planning lookup in inspected v1 routes. Do not port legacy code automatically.
External market references consulted 2026-11-25: [Meey ecosystem](https://meeygroup.com/he-sinh-thai), [Meey Map data FAQ](https://help-center.meeymap.com/cau-hoi-thuong-gap-meey-map/cau-hoi-ve-du-lieu-tren-meey-map), [Guland terms](https://guland.vn/dieu-khoan-thoa-thuan), [Batdongsan.com.vn listing rules](https://trogiup.batdongsan.com.vn/docs/quy-dinh-dang-tin-bat-dong-san). Evidence of available offerings and their operational concerns, not independent proof of demand, profitability or BDSPro advantage. No data access rights or current dataset verified.

## Challenge and next decision
Separate ecosystem aspiration from first value proposition. Candidate A: planning information first; B: private property/listing management first; C: public listing marketplace first; D: limited connection between property record and planning evidence if both are available and real workflow warrants it. No selection accepted.
Planning first requires a known data source, permitted use, sample coverage, version/date/provenance, handling missing/conflicting data, update owner and a real lookup task. A lookup is reference information, not guaranteed legal clearance. Listing first requires real listing workflow and visibility/ownership; marketplace adds supply/demand and moderation. No assumption of AI/chat/CRM/payment as prerequisites.
First acceptance candidate: a user can find the intended parcel/location in scoped data, see the relevant source/date/coverage or explicit no-data/uncertainty and know how to verify; test with approved reference examples, not invented targets. Adoption, payment, enterprise use and actual transactions are later signals as applicable, not all first-release gates.
Resolve: why developer prefers planning (actual business instruction, available data, observed need, or hypothesis); whether current listing/property work is a firm delivery commitment; data availability and lawful access; initial public/private value and measurable outcome. Unknown is valid. Keep topic 01 OPEN; no architecture/runtime changes, no customer claims or production proof.
