# Organization, Team, Group, Community, Channel & Provider Participation — 2026-10-09 follow-up

**Checkpoint status:** semantic BUSINESS INTENT agreed for further core review; physical schema and most data policies NOT CLOSED.  
**Source:** user's 2026-10-09 discussion following enterprise-first onboarding and managed-identity review.  
**Branch:** `architecture/canonical-database-final`, docs-only. DBML/Go/migrations unchanged.

## One operating story

ABC is a legally recognized real-estate organization using BDSPro as a business platform. It onboards newcomer An, who has **no pre-existing personal BDSPro account**. ABC wants to supply access, membership, working context, assigned team, CRM pipeline, approved listing actions and financial/seat cost allocation; it also wants to offboard An without destroying ABC's operational records or An's enduring human identity. **Enterprise-first onboarding is a FIRST-CLASS business use case**; corporate provisioning or employer-owned credentials are optional implementations with additional lifecycle/control cost, not mandatory for every enterprise-first newcomer.

ABC has branches by location, internal departments/teams for functional responsibilities, and potentially per-team private channels. ABC also participates in independent cross-company collaboration groups or co-founded groups for distribution of supply/projects, and operates/joins community discussions for local buyers, sellers and professionals. A person may be present in many contexts and retain identity and earned, attributable career recognition without cross-tenant CRM leakage.

ABC posts a verified property via an employee who acts as publisher **on behalf of** ABC, or a private owner lists their own home, or a licensed professional provides brokerage services under an eligible business. These are different legal/authority contexts, NOT mutually exclusive Person types or subscription levels. The same person may buy, sell own property, be a professional and participate in several organizations over time.

## Semantic taxonomy (NO table creation yet)

| Kind | One-row business assertion if independently durable | Boundary and warning |
| --- | --- | --- |
| **Organization** | independently identifiable legal/business organization | Party subtype; not an Account, team or tenant admin credential. Corporate legal identity != subscription buyer != workspace owner. |
| **Branch** | organization's identifiable operational location/subdivision with branch-scoped responsibilities | Not automatically separate Party; branch may be a geographic business unit; same-org invariant for staff. |
| **Department** | relatively enduring organization function (sales/CRM/legal) with management/reporting and work scope | May need entity only when independent hierarchy or policies matter; not automatically a branch. |
| **Team** | operating set of people for shared work, tasks, lead routing or permissions | A team can span branches if business allows. Team membership != organization membership, and team existence != independent legal identity. |
| **Class** | ambiguous UI/business vocabulary: if small staff cohort, map to Team; if training cohort, independent learning lifecycle may be justified | Never create generic `classes` without concrete instructor/student/curriculum/authority lifecycle. |
| **Group** | collaboration network with goal; may be internal or cross-organization and may be independent of one sponsor | Size is not the definition; do NOT assume all groups are descendants of Organization or authorized to read its private data. |
| **Community** | interest/place/content/social network with audience and moderation | Membership is not employment, organization role or verified brokerage standing. |
| **Channel** | communication/distribution surface within a defined context (team/group/community/org) | Channel subscriptions/views do not themselves confer legal identity, CRM access or business authorization. Need distinct use case for private messaging vs marketing feed vs lead distribution. |
| **Deal** | transactional collaboration, contract and economic responsibilities in specific scope | Participation/leadership != legal party to a contract; may span several organizations. |
| **Provider/Seller** | action capability and role for a specific selling/letting/service situation | Not a Person subtype; party publishing Listing != property owner != broker != authorized representative. |
| **Membership/Role/Authority** | relation of a person to context, assigned permission, and proof of delegated/legal capacity | Distinguish these facts; hierarchy alone never automatically grants all underlying access. |

### Internal and external topology (conceptual, NOT FK schema)

```text
LEGAL ORGANIZATION ABC (Party)
  |-- branch Hanoi     -- geographic responsibility
  |-- branch HCMC
  |-- internal sales team, CRM team, project team (potentially cross-branch)
  |-- internal channels / reporting
  |-- business listings, restricted CRM, assigned deals

INDEPENDENT OR PARTNER NETWORK
  |-- collaborative group: ABC + XYZ + independent qualified professionals
  |-- local real-estate community: buyers, owners, companies, contributors
  |-- communications/distribution channels under an explicitly selected context
  |-- deal with specified participants, obligations and permissions

An: one durable Person; joins ABC from enterprise-first onboarding,
can participate in team/group/community/deal under DIFFERENT permission policies.
```

## Real comparisons, appropriately bounded

- GitHub Organization -> Teams, optionally nested, scope resource permissions within organization; GitHub teams are organization-bound, not the BDSPro model for **external collaboration networks**: https://docs.github.com/en/organizations/organizing-members-into-teams/about-teams
- Microsoft Teams -> Team + standard/private/shared Channels; shared channels can enable cross-team/external collaboration; this distinguishes internal work team from communication surface: https://learn.microsoft.com/en-gb/MicrosoftTeams/teams-channels-overview
- HubSpot CRM team-based record access limits access to owner/team, not merely membership in same company: https://knowledge.hubspot.com/records/assign-access-to-records
- In Vietnam, brokerage business and individual professionals have separate legal conditions under Article 61, Law on Real Estate Business 2023; confirm amendments and applicability before enforcing policies: https://xaydungchinhsach.chinhphu.vn/tu-1-1-2025-moi-gioi-bat-dong-san-khong-duoc-hanh-nghe-tu-do-119231222132743287.htm

These prove known solutions to concrete pressures, not that BDSPro should copy their number of tables or inherit permissions.

## Provider / Seller / Publisher decision dimensions

For **each action**, establish:
1. **ACTOR**: Account/session/service principal actually calls API. Enterprise-first employee may have BDSPro Account with organization-issued invitation or corporate SSO; provisioned work identity if earned.
2. **REPRESENTED PARTY**: Person acting personally OR Organization via valid membership/authority.
3. **CAPACITY**: personal asset owner, legally authorized representative, appropriately qualified broker/professional, company business representative, or non-selling community participant.
4. **RESOURCE**: precise Property/Listing/Lead/Deal/CRM record and its owner/tenant.
5. **PROOF**: required permission to publish/offer/represent that specific asset and current verification/qualification state; no general "broker badge" substitutes for permission on a property.
6. **OPERATION**: draft, submit for review, publish, distribute, handle inquiry, negotiate, contract, bill/settle each have **different risk gates**.
7. **BENEFIT**: subscription/tools/paid promotional placement and reputational recognition are separate.

Examples:
- Self-owner Person A offers home: account+Person A+property-specific evidence+publish policy; no brokerage assumption merely due to sale.
- Employee An publishes ABC listing: Account An -> organization context ABC -> active Membership -> scoped team/role -> ABC offering/publishing authority -> policy/evidence; listed publisher ABC may differ from human action actor.
- Qualified broker collaborates in Group with ABC/XYZ: participation in Group alone cannot publish on behalf of any firm, reveal CRM or create commission entitlement.
- Enterprise pays promotion: entitlement buys exposure, NOT verified ownership or organic top rank.

Trusted recognition: verified badges attest specific aspects; reputation performance needs evidence/sample size/policy version; no new-user prejudice from missing sample; incentives capped; no financial/legal/security privileges from tier. Previous 60/40, 180 days and trial level thresholds remain experimental ONLY.

## Permission and data-scope invariants (PROPOSED)

- Organization membership **does not grant** branch/team/CRM data blanket access.
- Team/group/channel participation cannot bypass protected-record ownership/policy.
- Team scope must belong to intended Organization when it is an internal team; explicit cross-org collaboration modeled separately.
- Person role on Group != authority to act as Organization Party.
- Group creation is not organization legal registration; Group may have multiple affiliated/sponsor organizations and persons, while legal signatories on Deal remain explicit.
- Deal participants must not view another party's private CRM or confidential commission terms by default.
- Entering corporate first does not require a previous consumer onboarding journey.
- Revoking enterprise access must preserve necessary business records and avoid transferring protected data to a departing employee or vice versa.
- Network discovery/public profiles/SEO cannot use private CRM leads without lawful purpose and permission.
- Source of listing: publication approval must identify actor, represented Party, authority for property and evidence version applicable at publish time.

## Review of current 146-table candidate

Canonical currently:
- `organization_branches` and `organization_branch_assignments` exist; **no explicit department/team table**, and no explicit hierarchy/class/channel type for enterprise division.
- `groups` has standalone id/name/description and `group_participations.person_id`; lacks explicit sponsor/org affiliation or cross-company Group membership model.
- `communities` and `community_memberships` cover distinct social audience but moderation/enterprise partnerships are still to review.
- `deals` has `organization_id`, `group_id`, `branch_id`; semantics for collaboration spanning multiple companies and financial/legal party roles need verification.
- `listings.publisher_party_id` is Party, which distinguishes published business identity but **not** actual actor, beneficial owner, brokerage status, delegated right to publish, or listing review evidence.
- `organization_roles`/assignments exist, but role/branch/membership same-org integrity must be proven; no automatic claim that team hierarchy or Group confers protected permissions.
- `subscriptions.subscriber_party_id` maps to Party; Group/Community not automatically Party; billing payer vs service beneficiary may differ.
- `crm_contacts.organization_id` is a tenant scope, not permission for everyone on shared Group or subscribed Channel.
- Do NOT rename arbitrary existing `groups` to `teams` or add `org_id` to Groups until a real internal-vs-external collaboration policy is derived.

**Changes to evaluate** via minimum counterexamples, not automatic migrations:
- Internal Team/Department relation and membership (maybe branch associations) and scopes; role vs lead routing/functional assignment distinction.
- Group sponsor/network participation by Organization and/or Person, who owns shared deliverables and when an org becomes a contracting Party.
- Channel audience/ownership, cross-org participation, moderation and privacy, separate from Group/Community.
- Provider onboarding, verified legal capacity and specific listing publication authority.
- Enterprise-invited newcomer vs pre-existing person, business-controlled credentials and offboarding.
- Reputation attribution to Person vs Organization vs Team/Group (avoid automatically transferring reviews).
- Commercial beneficiaries for paid team/group/community tools without promoting those spaces to legal Party.

## Value-vs-cost boundary

| Candidate | Buy real value when | Upfront / ongoing cost |
| --- | --- | --- |
| Team inside Organization | delegation, team-owned CRM, lead routing, visibility limits proven | members, team-scoped authorization, transfer/reassignment, incident/audit |
| Department | multi-team hierarchy and reporting/administrative accountability needed | hierarchy/moves, reporting consistency, permission inheritance risks |
| Channel | message/content delivery policies differ from owner Group or Team | per-channel ACL, privacy, moderation, retention, fanout |
| Cross-company Group | joint sourcing/distribution actually demanded | org+person representation, permission on shared assets, ownership/disputes/consent |
| Provider activation | transition from inquiry user to legal owner/representative/broker needed | KYC/qualification, appeal, fraud, publishing reviews |
| Managed identity | customer cannot be satisfied by invitation/SSO/roles and needs full work-credential lifecycle | provisioning/recovery, offboarding, identity fragmentation and support |

## Immediate continuation

ORG-01 remains first: Organization legal entity identity, registration/claim, representative authority and enterprise-first newcomer onboarding. Then a paired **ORG-NET-01** scenario:
1. ABC provisions An -> An becomes a member -> assign to internal Team Hanoi Sales -> Team sees its allocated CRM contacts, not all ABC contacts.
2. ABC and XYZ cooperate via a Group on a property; XYZ contractor can see precisely shared Listing/Deal data, NOT ABC's CRM.
3. An publishes on behalf of ABC; both An actor and ABC publisher/authority evidenced; cannot use a Group role to override publication permission.
4. An leaves ABC and joins XYZ; revoke ABC access, preserve legally necessary ABC CRM and An's own verified personal reputation without leakage.
5. Compare independent internal Team vs use existing organization_branch_assignments/role to see whether a new table is earned.

**Proposed review outcome:** identity taxonomy and scoped access rules PRINCIPLE. Team/Channel/Group affiliation physical tables OPEN. No production code or DBML changes authorized without explicit further human approval and lab evidence.
