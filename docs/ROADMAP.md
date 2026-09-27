# Sovereign Conquest: Product and implementation roadmap

- **Document version:** 00.02.01
- **Revision date:** 2026-09-27
- **Classification:** Repository integration and traceability correction; documentation only
- **Application baseline:** 01.06.03
- **Source assessment commit:** `03625c68a141f649aa87480b13d61527febceff1`
- **Implementation status:** Planned; importing this document does not complete any implementation packet

## Version and scope

This revision integrates the evaluated roadmap into the repository as the canonical
planning document. It preserves the accepted design from document 00.01.00 and the
codebase assessment and implementation plan from document 00.02.00. It adds navigation
and one packet status table, and corrects the scope, changelog, commit notes and
rollback instructions for repository maintenance. It introduces no new gameplay plan.

Document version 00.02.01 is separate from application version 01.06.03. This is a
documentation integration, not an application release, implementation milestone or
deployment. The source assessment and its validation results remain tied to commit
`03625c68a141f649aa87480b13d61527febceff1`, reviewed on 2026-09-27. They are historical
evidence from document 00.02.00, not new results for the integration commit.

Use the project's xx.xx.xx format: Release Version.Feature Update.Bug Fix.
Increase the document feature field for additive roadmap capabilities and reset the
fix field; increase the fix field for corrections, including this integration;
increase the release field for a new major baseline and reset the other fields.
The next additive document revision would be 00.03.00; a correction would be 00.02.02.

Use `roadmap-v00.02.01` if tagging this exact documentation revision. It is the
intended documentation tag, not a claim that a tag has been created. Game release
tags must follow the application's actual version history. Record actual integration
and implementation commit hashes and review links in Git history and the relevant
pull requests; do not invent them in this document.

Requirement labels SC-R01 through SC-R08, findings F01 through F11, and implementation
packets SC-I01 through SC-I15 are local roadmap references, not created issue tracker
tickets. Section 19 separately identifies dependency PRs observed during the source
assessment. The earlier phase grouping in Section 3 remains product context;
Section 16 governs execution on the existing codebase.

## Navigation

- [Product direction](#1-product-direction) and [gameplay requirements](#2-gameplay-requirements)
- [Phases](#3-revised-implementation-phases) and [playtest plan](#4-playtest-and-evaluation-plan)
- [Release requirements](#10-release-and-implementation-requirements), [document history](#11-changelog-and-release-notes), and [commit notes](#13-copyable-commit-notes)
- [Codebase assessment](#15-current-codebase-assessment) and [implementation plan](#16-dependency-ordered-implementation-plan)
- [Migration and rollout](#17-migration-compatibility-and-rollout-contract), [validation evidence](#18-validation-evidence-and-required-new-coverage), and [dependencies](#19-dependencies-release-traceability-and-open-work)
- [Next checkpoint](#20-next-implementation-checkpoint) and [original appendix](#appendix-a-supplied-unversioned-roadmap)

## Implementation tracker

This table is the single packet status register. Every packet starts as **Planned**.
Change a status only when there is supporting implementation and review evidence;
link the actual issue or pull request when one exists. Milestone versions and
acceptance gates remain defined in Section 16. Dependencies below summarize each
packet's full conditions and do not replace them. SC-I14 has an M5 noncompetitive
pilot and an M6 scored trial; neither alone completes the whole packet.

| Packet | Milestone | Dependencies | Status |
| --- | --- | --- | --- |
| [SC-I01. Establish a database-backed baseline and ordered migrations](#sc-i01-establish-a-database-backed-baseline-and-ordered-migrations) | M1 | Reviewed application baseline | Planned |
| [SC-I02. Repair the economy, progression, chat, and rejection paths](#sc-i02-repair-the-economy-progression-chat-and-rejection-paths) | M1 | SC-I01 test harness | Planned |
| [SC-I03. Enforce account revocation and consistent reset/spawn state](#sc-i03-enforce-account-revocation-and-consistent-resetspawn-state) | M1 | SC-I01 | Planned |
| [SC-I04. Make release promotion and production configuration trustworthy](#sc-i04-make-release-promotion-and-production-configuration-trustworthy) | M1 | Preceding M1 checks (SC-I01-SC-I03) | Planned |
| [SC-I05. Add command receipts, economic journals, and typed events](#sc-i05-add-command-receipts-economic-journals-and-typed-events) | M2 | M1 | Planned |
| [SC-I06. Persist scheduled work and centralize balance rules](#sc-i06-persist-scheduled-work-and-centralize-balance-rules) | M2 | SC-I05; ordered migrations | Planned |
| [SC-I07. Consolidate the browser contract, then add structured actions and map](#sc-i07-consolidate-the-browser-contract-then-add-structured-actions-and-map) | M3 | SC-I05; parallel with SC-I06 | Planned |
| [SC-I08. Extend intel, practical loadouts, and corporation permissions](#sc-i08-extend-intel-practical-loadouts-and-corporation-permissions) | M3 | SC-I05, SC-I06; UI also SC-I07 | Planned |
| [SC-I09. Implement one freight state machine for NPC and player contracts](#sc-i09-implement-one-freight-state-machine-for-npc-and-player-contracts) | M4 | SC-I05, SC-I06, SC-I08 | Planned |
| [SC-I10. Deliver the first expedition and a useful return screen](#sc-i10-deliver-the-first-expedition-and-a-useful-return-screen) | M4 | SC-I07, SC-I09 | Planned |
| [SC-I11. Implement exposure, encounters, retreat, and recovery before losses](#sc-i11-implement-exposure-encounters-retreat-and-recovery-before-losses) | M5 | SC-I05-SC-I10 | Planned |
| [SC-I12. Add bounded escorts and relay supply/control](#sc-i12-add-bounded-escorts-and-relay-supplycontrol) | M5 | SC-I09, SC-I11 | Planned |
| [SC-I13. Replace soft wipe with an audited season transition](#sc-i13-replace-soft-wipe-with-an-audited-season-transition) | M6 | SC-I05, SC-I06, SC-I12 | Planned |
| [SC-I14. Run the complete multiplayer pilot and tune from evidence](#sc-i14-run-the-complete-multiplayer-pilot-and-tune-from-evidence) | M6; M5 pilot | M5 for noncompetitive pilot; M6 for scored trial | Planned |
| [SC-I15. Expand only after the complete pilot supports it](#sc-i15-expand-only-after-the-complete-pilot-supports-it) | M7 | SC-I13, SC-I14 | Planned |

## 1. PRODUCT DIRECTION

Sovereign Conquest is an asynchronous, turn-based space strategy game in which
logistics, intelligence, and territorial ambition make players consequential
to one another. Preserve the Trade Wars character: a shared sector graph,
limited turns, local commerce, dangerous routes, deployed defenses, planets,
corporations, and decisions driven by resources and information.

The central experience:
Discover an opportunity, choose a risk, trade or cooperate to pursue it, change
the shared world, and leave with a specific next ambition.

Design for three reasons to return:

* Mastery: the player can make better decisions through experience and scouting.
* Agency: several viable plans exist, each with costs and consequences.
* Belonging: other people value the player's knowledge, reliability, or support.

A successful first-week story might be: our scout found an overlooked approach,
our traders financed a shipment, our escort got it through a blockade, and our
corporation established a foothold that changed a rival's plans.

Enjoyment and voluntary return are the goals. Players must have satisfying
stopping points and a reasonable way to recover from absence or defeat.

## 2. GAMEPLAY REQUIREMENTS

### SC-R01. CONNECT COMMERCE, SUPPLY, AND WARFARE

Keep three initial commodities: ore, organics, and equipment. Give each actual
uses as well as resale value:

* Ore supports construction and repairs.
* Organics support colony operation and growth.
* Equipment supports defenses, relay operations, and expansion.

The prototype only needs a small subset of these consumption rules, centered
on one shared supply objective. Broader colony simulation belongs later.

Ports buy and sell at local prices affected by stock, replenishment, and real
consumption. Scheduled replenishment remains a baseline mechanic, but must not
make the same route permanently optimal. Commodity quantities, price bounds,
and replenishment behavior must be visible enough for players to plan.

Transport connects regional production and demand. Blockades interrupt
deliveries; shortages create commercial opportunities; successful deliveries
help sustain defenses or contest an objective. Actual supply consequences must
make warfare relevant to traders and logistics relevant to combat players.

Example: a corporation needs equipment at a relay. Its usual route is blocked.
A scout discovers another approach, a hauler accepts an escrow-backed delivery,
and an escort commits protection. Completion changes the relay's operating
state and advances the corporation's objective.

Track separate economic flows:

* Currency creation, transfers between players, and currency removal.
* Commodity production, consumption, transfer, and destruction.
* Asset construction, repairs, losses, and replacement assistance.

A payment to another player is a transfer, not a currency sink. Use bounded NPC
rewards, construction costs, repair costs, and appropriate upkeep to support a
working economy without creating an unlimited money or resource source. Tune
from observed activity; do not introduce a fourth commodity before the initial
three create worthwhile decisions.

### SC-R02. MAKE PLAYERS USEFUL BEFORE THEY BECOME WEALTHY

Start with three equipment specializations, not permanent character classes:

* Scout: better reconnaissance, route knowledge, and information gathering,
  traded against cargo capacity or direct combat capability.
* Hauler: efficient cargo movement, traded against combat or scouting capacity.
* Escort: protective capability, traded against hauling or independent profit.

Allow affordable switching so a first-session choice does not lock a player
into an unwanted career. Make small ships useful through situational strengths
and limited objectives. Additional industrial or diplomatic specializations
can follow when the first three have proven useful.

Move basic corporations into the first multiplayer implementation phase:

* Corporation creation, membership, recruitment, and chat.
* Shared objectives, a small treasury, shared assets, and role permissions.
* Selective sharing of scans and route intelligence.
* Simple delivery contracts with escrow, explicit completion conditions,
  deadlines, cancellation rules, and visible consequences of failure.
* Activity and contribution records that acknowledge noncombat work.

Cooperation must work asynchronously. For the prototype, escort support should
be a precommitted defensive assignment attached to a qualifying convoy or
mission and resolved by authoritative server rules. Define its duration, turn
and resource costs, support limits, cancellation, and exposure to loss before
implementation. An escort must not protect unlimited simultaneous missions or
depend on both players being online together.

Example: a scout leaves a timestamped route report; a trader accepts the supply
job later; an escort has already committed bounded protection. Each participant
can contribute during a different session.

Retain diplomacy as a product direction. Simple public standings, explicit
agreements, and dependable permissions come before elaborate treaty systems.
Define which agreements the game enforces and which are social promises.

### SC-R03. KEEP STRATEGIC CHOICES INTERESTING

Create competing advantages rather than a single universal upgrade path:

* Profitable shortcut versus safer detour.
* More cargo versus stronger protection.
* Private exploitation of a discovery versus valuable cooperation.
* Immediate profit versus financing a shared objective.
* Concentrated defenses versus protecting several vulnerable routes.

Preserve per-player fog of war. Store discovered geography separately from
time-sensitive observations. Every scan or shared report needs an observation
time and source. Show stale information as stale; do not silently present an
old scan as the current state of a sector.

Information can change decisions without exposing everything. Route suggestions
and market analytics must only use information the player is authorized to
know. They must not reveal undiscovered sectors, hidden defenses, or private
corporation data. Preserve discovery while reducing repetitive clicks.

Combat remains resource- and intelligence-driven, not dependent on reflexes.
Retain sector-fighter combat, ship combat, scouting, defensive staging, and
eventually mines and traps. Include objectives short of total destruction:
intercept a delivery, break a blockade, force a retreat, or scout an approach.

Before committing, show known turn costs, resource costs, and risks. Distinguish
known facts from uncertainty. After resolution, explain the important factors,
losses, retreat or escape behavior, and possible counterplay without exposing
information the player has not earned.

Example: a smaller group successfully interrupts a valuable shipment without
having to destroy the larger corporation's entire fleet or capture its planet.

### SC-R04. DESIGN THE FIRST SESSION AND THREE SCALES OF GOAL

Prototype an opening expedition lasting roughly twenty minutes:
1. Create a pilot and receive a useful starter ship.
2. Accept a bounded introductory freight job.
3. Discover two ports and learn movement, scanning, cargo, and pricing through
   play rather than a long prerequisite manual.
4. Choose between routes with clearly explained differences in risk and reward.
5. Complete a profitable delivery and understand why it succeeded.
6. Buy an upgrade that changes a capability or tradeoff for the next expedition.
7. Reveal a concrete next opportunity and a safe place to finish the session.

Maintain three goal scales:

* This session: a profitable delivery, discovery, repair, or useful contribution.
* This week: a ship configuration, foothold, relationship, or personal project.
* This season: a corporation ambition and an attainable personal distinction.

Lead with structured panels for maps, ports, cargo, objectives, and corporation
activity. Keep the terminal command pane available for veterans. Both modes
must issue the same authoritative commands and show equivalent costs and risks.

The return screen answers: what changed, why it matters to my plans, and what I
can do next. Prioritize relevant changes over an unfiltered wall of logs.

Example: a returning player sees that a known route is blocked, a corporation
delivery pays more than yesterday, and a recently shared scan suggests a detour.

### SC-R05. SUPPORT USEFUL SESSIONS, ORDINARY ABSENCE, AND RECOVERY

Keep regenerating turns with a cap. One turn every two minutes is a starting
hypothesis, not a settled balance value. At that rate, 720 turns regenerate per
day. A cap of 100 fills in three hours and twenty minutes; a cap of 720 fills in
twenty-four hours from empty.

Prototype capacity for at least a full day of regeneration, then measure whether
ordinary work and sleep schedules cause meaningful disadvantage. Choose action
costs, expedition length, and capacity together. A full day of stored turns must
not require hundreds of repetitive commands to spend usefully.

Initial experience targets to test:

* A useful five-minute visit for reviewing changes or making a contribution.
* A satisfying twenty-minute expedition with meaningful decisions.
* A clear stopping point: dock, settle proceeds, leave information or bounded
  standing orders, and identify the next goal.

Define the offline rules before territorial PvP opens:

* Safe docking locations and their limits.
* What happens to ships and cargo exposed outside safe docking.
* Advance notice and scheduled vulnerability windows for major infrastructure.
* Persistent defense orders and how they resolve without the owner online.
* What assets can be lost, recovered, or protected.

Docking must not provide an instant escape from an already committed encounter.
Attack-window changes require notice and stable rules. Notifications can inform
players through chosen alerts or digests; they must not become a requirement to
answer overnight defense alarms.

Provide a basic replacement ship and accessible earning work after defeat.
Bound replacement aid so deliberate losses or alternate accounts cannot turn it
into a profitable resource source. Preserve a credible restart without erasing
the consequences that make expeditions and territory valuable.

Example: a player loses an exposed shipment, reads what went wrong, resumes
earning with a basic ship, and plans a different route without losing the whole
reason to participate.

### SC-R06. KEEP SMALL GROUPS AND LATE ARRIVALS COMPETITIVE

Expansion creates longer supply lines, upkeep, and more places to defend.
Avoid unlimited passive income from conquered planets. Distribute valuable
objectives so one concentrated fleet cannot defend every important location.

Small groups need useful limited operations, and new pilots must contribute
through scouting, deliveries, and support before acquiring expensive ships.
Offer protected starting space with worthwhile introductory work. Prevent it
from becoming a consequence-free staging area for veteran attacks.

Corporation size limits alone cannot stop dominance; corporations can form
alliances. Test coordinated groups, control of multiple objectives, alternate
accounts, repeated beginner targeting, contract collusion, and score farming.
Use clear rules, auditable evidence, and proportionate intervention. Do not
treat a shared household or IP address alone as proof of abuse.

Competitive monetization must not sell additional turns, stronger ships, or
superior intelligence. Cosmetics and presentation options may be explored later
without changing competitive capabilities or access to necessary information.

Example: a late-arriving scout uncovers a weakly watched route and earns a role
in a corporation's operation without first matching veteran wealth.

### SC-R07. DEFINE CONQUEST, SEASONS, AND LASTING IDENTITY

Prototype a competitive season of approximately eight weeks. This duration is a
playtest hypothesis. An early pilot can use a shorter declared test interval
without establishing the production season length.

Start with one type of territorial objective: corporations supply and defend
strategically distributed relay stations. Phase 1 needs only a minimal instance
of this contest; later seasonal testing adds distributed objectives and scoring.

Publish the scoring rules, finish time, eligible actions, and tie rules before
competition begins. Score verified objective outcomes rather than raw transfer
volume. Supplies must meet a bounded objective need; cycling goods between
friendly accounts must not manufacture points. Keep the scoring formula and
control/upkeep relationship simple enough for players to understand.

Recognize exploration, commerce, and defense alongside the corporation result.
Personal distinctions should remain achievable without belonging to the largest
group, and should not award permanent competitive power in later seasons.

Publish a reset contract before players commit:

* Reset seasonal territory, competitive wealth and fleets, resource stockpiles,
  and map intelligence according to the declared season rules.
* Preserve account identity, corporation identity where appropriate, names,
  medals, historical records, and cosmetic distinctions.
* Archive each season's world and event history separately from the active game.
* Do not carry secret old-map knowledge into a newly generated season as though
  it were current intelligence.

Name ships and owned locations where permitted. Record landmark discoveries,
ownership changes, major deliveries, defenses, and battles. Show relevant
history on maps and reports so participants can recognize their contribution.
Archives must preserve the game's privacy and intelligence rules; a public
chronicle does not automatically expose private chat, trades, or hidden scans.

Example: a corporation loses its seasonal territory at the reset but retains
its identity and the record of the supply operation that decided a major siege.

### SC-R08. MAKE A SMALL POPULATION FEEL CONSEQUENTIAL

Begin with one compact, populated universe. Use shared ports, delivery jobs,
and contested routes to create encounters and cooperation. Do not fragment the
early audience across many public servers or competing rule configurations.

NPC freight jobs, pirates, and limited exploration opportunities support quiet
periods. Bound NPC rewards and review their effect on player commerce so they
do not become an unlimited substitute for dealing with other players.

Exploration must reveal opportunities that change plans: an overlooked warp,
a damaged station worth restoring, an unusual resource opportunity, or a route
with a different risk profile. Use unfamiliar situations within understandable
rules; a changed rule must be explained.

Generate a reproducible sector graph from a seed and store it as canonical.
Discovering existing sectors does not regenerate the graph. If population
warrants growth, append new sectors and connections through an explicit,
recorded expansion event. Preserve existing IDs, geography, ownership, and
history. Use a separately seeded graph for a new season where declared.

Optional server configurations remain a future capability. Validate the core
experience in one universe before offering many public variants.

Example: twelve pilots can meet around a few meaningful trade routes and one
shared contest, while a later population increase opens a documented frontier.

## 3. REVISED IMPLEMENTATION PHASES

These are the accepted product-design groupings. The codebase already contains
parts of all three phases. Section 15 records that baseline; Section 16 is the
authoritative implementation order for extending the existing application.

### Phase 0: Define the playable rules and test assumptions

Specify the first freight expedition, action costs, turn capacity, local price
rules, three ship tradeoffs, starter protection, defeat recovery, and one supply
objective. Write the docking, escort, contract, and scoring rules before those
mechanics can expose players to irreversible losses. Identify open balance
values explicitly instead of hiding them in implementation defaults.

Exit evidence: a designer or developer can walk through a trade, a shared
delivery, a contested objective, a defeat, and a return session without inventing
missing rules along the way.

### Phase 1: Build a small, complete multiplayer experience

Preserve the original playable foundation and bring forward its social purpose:

* Login, pilot creation, useful starter ship, protected introductory work.
* Sector movement, scanning, per-player discovery, and a basic map.
* Ports with buy/sell, local stock-sensitive prices, and bounded replenishment.
* Turn regeneration, capacity, explicit action costs, and session previews.
* Scout, hauler, and escort equipment tradeoffs.
* Basic corporations, chat, shared treasury/assets, permissions, and intel.
* One delivery contract type with escrow and one bounded escort assignment.
* A minimal contested supply objective using a relay, with clear consequences.
* Simple sector-fighter and ship combat under defined protection rules.
* Safe docking, understandable defeat, replacement access, and earning recovery.
* Structured panels, optional terminal commands, useful logs and event feed.
* First-session guidance and a return screen that surfaces relevant changes.

Build order and purpose:
1. Guided freight expedition and meaningful upgrade: test the first next goal.
2. Responsive local markets and specializations: test interesting choices.
3. Corporations, contracts, and shared intel: test usefulness to other people.
4. One contested supply objective: test commerce producing strategy and conflict.
5. Safe stopping and recovery: test return after absence and setbacks.
6. Clear event history and return screen: test understanding and attachment.

Protection and recovery are release prerequisites wherever earlier steps expose
players to loss; their placement above is not permission to ship unsafe PvP.

Exit evidence: players can complete the entire discover/trade/cooperate/contest/
recover cycle and independently identify another worthwhile action. Technical
completion alone does not satisfy this phase.

### Phase 2: Test sustained competition and deepen the shared world

* Planets, production, supply consumption, upkeep, and initial citadel defenses.
* Several strategically distributed relays and transparent seasonal scoring.
* Published season/reset rules, personal distinctions, and historical archives.
* Stronger diplomacy and corporation asset management based on observed needs.
* Mines, traps, and counterplay that expand choices without obscuring outcomes.
* Measured adjustments to late-entry prospects, expansion costs, and recovery.
* Better contribution recognition and meaningful player/location identity.

Exit evidence: smaller groups remain useful, leaders remain contestable, players
understand losses and resets, and the economy supports repeated varied choices.

### Phase 3: Expand proven systems

* More sophisticated citadels, industrial options, and exploration situations.
* Invasions, anomalies, and limited-time sectors with clear rules and sufficient
  participation windows for the intended audience.
* Market analytics and route suggestions constrained by discovery and intel.
* Dedicated mobile interaction improvements and responsive layout refinement.
* Canonical frontier expansion and optional server configurations when audience
  size supports them.
* Additional commodities only when they add decisions the first three cannot.

Make basic layouts responsive and usable from the first client implementation.
Advanced mobile optimization can follow. Defer elaborate invasions, comprehensive
analytics, and deep citadel complexity until the complete early experience works.

## 4. PLAYTEST AND EVALUATION PLAN

First observe a few new players attempting the introductory expedition without
coaching. Note where they misunderstand prices, turns, risk, the interface, or
the next goal. Include people unfamiliar with terminal-based games.

Then recruit approximately 12-20 people into one compact multiplayer universe.
Include newcomers, solo participants, small groups, and people with limited or
different schedules. Use a clearly announced short pilot interval and reset
policy. Do not represent this sample as a reliable statistical retention study.

Record:

* Introductory expedition completion and time to a self-chosen next goal.
* Meaningful decisions versus repetitive commands during an expedition.
* Voluntary returns and each returning player's stated reason.
* Contracts, intel sharing, contributions, and helpful interactions.
* Recovery after defeat and whether the player resumes useful participation.
* Missed-turn capacity, required checking, and effects of ordinary absence.
* Wealth/territory concentration, new-player participation, and contested goals.
* Resource and currency flows, prices, supply shortages, and dominant routes.
* Reasons for leaving, including confusion, repetition, unfairness, or obligation.

Ask: What are you planning to do next, and what might stop you?
Ask: What specifically brought you back?
Ask: What made another visit feel unappealing or burdensome?

Answers naming a rival, discovery, shipment, counterstrategy, or shared project
are useful evidence that the intended experience exists. Answers limited to
waiting for turns or buying a larger number indicate a design problem to fix.

At larger scale, define cohort-based return measures such as day-one and
day-seven return, with documented windows and eligible populations. Distinguish
automated activity from human participation and compare results with qualitative
feedback. Avoid arbitrary industry retention targets or claims that time logged
in proves enjoyment. Collect only information needed to improve the game.

## 5. WEB ARCHITECTURE

Client

* Single-page web app with structured panels and an optional command pane.
* Map, cargo, ports, objectives, corporation roster, and sector intelligence.
* Accessible controls, readable costs, mobile-aware layouts, and clear feedback.

Authoritative server

* REST or GraphQL for authorized state reads; choose one primary read style.
* POST /command for state-changing commands, or a validated equivalent over
  WebSocket. Do not maintain divergent game rules across transports.
* Existing workers handle ports, planets, events and Protectorate state. Preserve
  on-demand turn regeneration during locked player reads/commands. Extend durable
  jobs for standing orders, attack windows and seasons as those features arrive.
* PostgreSQL for canonical persistent state and transactional game operations.
* WebSockets or server-sent events for combat outcomes, relevant market changes,
  corporation chat, objective status, and sector activity alerts.

Pushed events make the game responsive without requiring real-time reflexes.
Subscriptions must enforce the same visibility rules as ordinary state reads.
Use relevant summaries and player notification preferences to control noise.

## 6. DATA MODELING

The following is the original conceptual model; Section 15 identifies actual
current tables/modules. Extend the reviewed schema rather than creating duplicate
equivalents for concepts already implemented.

Retain the original core entities:

* users
* players: ship state, credits, turns, current sector
* sectors: attributes, optional port/planets, hazards, ownership, artifacts
* warps: origin, destination, directionality
* ports: type, inventory, local pricing inputs, replenishment schedule
* planets: owner, production, defenses, storage
* deployables: fighters, mines, limpets, and related artifacts
* corporations, corp_members, corp_assets
* logs: movement, combat, trades, and intelligence activity

Add or separate entities as implementation requires:

* player_discoveries and intel_reports, including observation time, source,
  visibility, and sharing permissions; discovery is not a global sector flag.
* ship_loadouts and escort_assignments with enforceable resource commitments.
* contracts, contract_events, and escrow/treasury ledger entries.
* relays/objectives, supply deliveries, ownership/control, and upkeep state.
* docking/protection state, attack declarations, windows, and defense orders.
* seasons, scoring_events, achievements, and historical world/event references.
* world_expansions, recorded seeds, and canonical graph-generation metadata.

These are logical modeling requirements, not an instruction to create every
table in Phase 1. Keep credits, escrow, inventory, turns, and scoring auditable.
Separate private operational logs from public chronicles. Make season boundaries
explicit in keys and queries so resets cannot mix old and current world state.

## 7. COMMAND ENGINE AND CONSISTENCY

Model gameplay as commands that transform authoritative state. Preserve existing
MOVE, SCAN, TRADE, PLANET, CORP, MINE, SHIPYARD and informational commands.
Ship attacks and deployable fighter commands are new M5 work, not existing code.
Introduce contract, delivery, escort, intel-sharing, docking, relay-supply and
attack-declaration commands only as their implementation packet requires.

Every state-changing command:
1. Authenticates the actor and validates authority and visibility.
2. Loads relevant state with appropriate transactional locking.
3. Checks costs, limits, timing, eligibility, and all preconditions.
4. Applies state changes atomically, including turns and resource transfers.
5. Commits an auditable result and publishes authorized events after commit.

Make retries and background jobs idempotent. Repeated requests must not double
spend, duplicate escrow payouts, regenerate the same turns twice, or award the
same delivery score again. Specify locking order for interactions involving
multiple accounts/assets. Use a durable event-publication mechanism where needed
so clients do not receive outcomes for a rolled-back transaction.

Keep deterministic rules testable. Graph generation must be reproducible, while
each recorded world remains canonical. Version rule/configuration changes and
retain the inputs needed to explain consequential outcomes.

## 8. SECURITY AND ABUSE CONTROLS

Preserve server-side validation, atomic turn consumption, activity logging, rate
limits, signed bearer sessions and the existing transport hardening. Current
authentication uses bearer JWTs, not cookies; add HttpOnly/Secure cookies and CSRF
protection only if adopting a separately planned cookie transport. Close the
session-revocation gap in Section 16. Never trust client-provided outcomes,
balances, or ownership.

Review authentication and authorization for every command and subscription.
Enforce corporation roles, escrow ownership, selective intel sharing, docking
eligibility, attack timing, and season boundaries on the server. Validate and
bound names, chat, quantities, prices, routes, and order payloads.

Begin anti-bot controls with rate limits, cooldowns, and auditable heuristics.
Review botting and collusion against actual evidence. Record investigations and
provide a correction path for false positives. Add moderation/reporting needs
to the implementation plan for public chat and player-created names.

Protect secrets and private data. Do not place credentials, session tokens,
private chat, or hidden intelligence in public event streams or routine logs.
Use dependency and static security scanning when code and tools are available.

## 9. CONFIGURATION AND BALANCE REGISTER

Treat these as explicit, versioned settings with documented bounds, examples,
and migration behavior. Do not expose unrestricted public variants initially.

Setting: turn_regen_seconds
Starting hypothesis: 120 seconds, equivalent to 720 turns per day.
Use: replenish a pilot's action budget predictably.

Setting: turn_capacity
Starting hypothesis: at least one day of regeneration; 720 at the rate above.
Use: allow ordinary work and sleep without frequent mandatory checking.
Validation: tune with action costs and real expedition length; this is not a
guarantee of equal outcomes for every amount of time away.

Setting: action_turn_costs
Starting hypothesis: unresolved per command; establish during Phase 0.
Use: ensure scouting, hauling, and combat have clear opportunity costs.

Setting: local_market_parameters
Starting hypothesis: bounded inventory-sensitive local prices and replenishment.
Use: permit regional shortages and shifting routes without infinite arbitrage.

Setting: contract_and_escort_limits
Starting hypothesis: one simple delivery type and bounded committed protection.
Use: enable cooperation across different play schedules; prevent duplicate
commitments, unbounded payouts, and indefinite resource locking.

Setting: protection_and_attack_windows
Starting hypothesis: safe docking, advance notice, and declared major-asset
windows; exact durations and eligible zones must be decided before PvP release.
Use: make exposure and defense commitments understandable.

Setting: recovery_assistance
Starting hypothesis: useful basic replacement ship and bounded starter earnings.
Use: support participation after defeat without profitable intentional losses.

Setting: season_duration_and_scoring
Starting hypothesis: eight-week season, relay supply/control objective.
Use: give accumulation and cooperation a finite competitive purpose.
Validation: use a shorter labeled pilot if needed; publish scoring and reset
rules before production participation.

Setting: world_size_and_expansion
Starting hypothesis: compact shared graph sized to actual participants.
Use: meaningful player contact at low population, with recorded append-only
frontier expansion when warranted.

## 10. RELEASE AND IMPLEMENTATION REQUIREMENTS

Apply the following gates to future code changes. This document revision does
not establish that any application check has already passed.

1. Version increment
   Use the three-field project format and the actual application baseline.
   Align release artifacts and tags with the tested commit. Avoid untraceable
   builds. Keep documentation and application versions distinguishable.

2. Changelog
   Record what changed, why, and whether it is breaking, additive, or a fix.
   Reference real issues and commits when available; never fabricate them.

3. Automated tests
   Run the full unit, integration, and regression suites for application changes.
   Add behavior tests for new commands and rules. Prioritize transactional
   trading, concurrent turn spending, escrow lifecycle, permissions, idempotent
   jobs, stale intel visibility, escort commitments, docking/attack windows,
   recovery abuse, scoring, graph expansion, and season resets.

4. Static analysis and linting
   Run the repository's linters, format checks, and type checkers. Resolve
   warnings with structural or security implications before release.

5. Security review
   Review authentication, authorization, input validation, logging, dependency
   changes, and secrets. Run available SAST and dependency scanning. Include
   cross-corporation visibility and event-stream authorization in the review.

6. Dependency validation
   Confirm compatibility, inspect available vulnerability results, and rebuild
   lock files when needed. Record unresolved findings honestly.

7. Build validation
   Build in a fresh environment from declared dependencies and committed config.
   Do not rely on undeclared local packages or services.

8. Configuration validation
   Check environment variables, defaults, feature flags, worker schedules, and
   the balance register. Verify config transitions for active contracts,
   attacks, stored turns, and seasons; avoid silently changing committed terms.

9. Database migration review
   For schema changes, test forward migration and compatibility with supported
   versions. Test rollback where safe; document backup/restore or forward repair
   when a reverse migration would lose data. Preserve ledgers and event history.

10. Performance check
    For core logic, query, or I/O changes, test realistic concurrent commands,
    crowded sectors, market contention, regeneration bursts, contract workers,
    and event delivery. Record latency, throughput, lock contention, and query
    behavior against an explicit baseline and target workload.

11. Logging and observability
    Keep structured command IDs, outcomes, timings, and relevant actor/object
    references. Monitor failed commands, authorization denials, job delay,
    economic imbalances, and scoring anomalies. Verify alerts and metrics while
    protecting secrets and private player information.

12. Documentation
    Update README, command/API examples, configuration guidance, architecture
    diagrams where affected, and player help. Every feature or option needs an
    example and use case. Explain costs, exposure, resets, and recovery clearly.

13. Backward compatibility
    Review APIs, command payloads, configuration fields, saved games, event
    formats, and active commitments. Declare breaking changes and provide an
    explicit transition plan instead of relying on undocumented behavior.

14. Rollback plan
    Keep the previous deployable artifact, its config, and required backups.
    Verify the revert path before deployment. Define treatment of contracts,
    escrow, scores, and events created under the new release; avoid duplicate
    payouts or unjustified removal of player actions during recovery.

15. Release notes and artifacts
    Attach the tested version, player-visible changes, operator instructions,
    validation evidence, known limitations, and relevant build artifacts.

16. Commit notes
    Supply copyable notes describing purpose, behavior changes, tests, risk,
    compatibility, and rollback. Insert actual issue and commit references when
    available. A proposed commit message is not evidence of a commit.

## 11. CHANGELOG AND RELEASE NOTES

### 00.02.01 | 2026-09-27 | Repository integration and traceability correction

Fix: make docs/ROADMAP.md the canonical repository roadmap, with navigation and
one status/dependency table for all fifteen implementation packets. Preserve all
twenty substantive sections, eight requirements, eleven findings, fifteen work
packets and the original supplied appendix. Reason: keep accepted design and the
evaluated implementation plan reviewable alongside the application source.

Fix: distinguish this documentation integration from the historical 00.02.00
assessment and validation. Update current scope, commit notes and rollback guidance
without changing the application baseline or claiming implementation progress.

Breaking impact: none. Documentation only; no application API, runtime, schema,
dependency or configuration behavior changes. All implementation packets remain
Planned. Application version 01.06.03 and proposed milestone release reservations
are unchanged.

Source artifact: Sovereign_Conquest_Roadmap.txt, document version 00.02.00.
Repository artifact: docs/ROADMAP.md, document version 00.02.01.
Source assessment commit: 03625c68a141f649aa87480b13d61527febceff1.
Documentation tag convention: roadmap-v00.02.01; creation is recorded separately.

Historical release notes follow. Statements about files, commits, tags, checks and
rollbacks within the 00.02.00 and 00.01.00 entries describe those revisions only.
They do not describe the repository integration commit.

### 00.02.00 | 2026-09-27 | Current-code assessment and implementation plan

Additive: identify main commit 03625c68a141f649aa87480b13d61527febceff1 and
application 01.06.03 as the reviewed baseline, with file/function evidence.
Reason: implementation must reuse existing Go/PostgreSQL/browser capabilities.

Fix to planning assumptions: distinguish existing corporations, ships, timestamped
intel, routes, planets and season records from missing gameplay. Preserve the
browser compatibility fixes and actual bearer-session/lazy-regeneration design.

Additive: record eleven source-grounded findings, fifteen dependency-ordered
implementation packets, seven proposed application releases, migration families,
acceptance tests, deployment/rollback rules and maintenance PR references.
Reason: address economy, security, data and release prerequisites before new
logistics, conflict and competitive seasons.

Additive: record locally executed tests/build/static/dependency checks, measured
coverage, unavailable integration/container checks and separately observed
GitHub workflow results. Reason: distinguish verification from assumptions.

Breaking impact: documentation only. No application API, code, database, lockfile,
GitHub branch, release or deployment was modified. Future compatibility changes
and proposed application versions are identified in Sections 16-19.

Reviewed application commit: 03625c68a141f649aa87480b13d61527febceff1.
Implementation issues/commits: SC-I01-SC-I15 are proposed work references only.
Release artifact: Sovereign_Conquest_Roadmap.txt, document version 00.02.00.


### 00.01.00 | 2026-09-27 | First versioned revision

Additive: SC-R01 connects the three commodities, local prices, supply objectives,
and conflict. Reason: trade should influence the shared strategic situation.

Additive: SC-R02 establishes scout/hauler/escort tradeoffs, delivery escrow,
asynchronous support, and meaningful contributions. Reason: players need a role
before they accumulate veteran wealth.

Additive: SC-R03 adds timestamped intel, competing choices, limited combat
objectives, and explanatory reports. Reason: decisions must remain interesting
after basic routes and upgrades are understood.

Additive: SC-R04 specifies onboarding, three scales of ambition, structured
controls, and a useful return screen. Reason: each session needs an understandable
purpose and a next step.

Additive: SC-R05 defines pacing hypotheses, safe stopping, offline rules, and
recoverable defeat. Reason: participation should remain practical around
ordinary schedules and setbacks.

Additive: SC-R06 adds expansion commitments, useful late entry, fairness tests,
and competitive monetization constraints. Reason: early leaders must not remove
everyone else's reason to participate.

Additive: SC-R07 defines a relay-based seasonal prototype, reset contract,
personal recognition, and historical identity. Reason: conquest needs a purpose
and player contributions need lasting meaning.

Additive: SC-R08 addresses low population, bounded NPC activity, discoveries,
and recorded world expansion. Reason: early universes must feel inhabited and
growth must preserve existing geography and history.

Planning change: corporations move from Phase 2 into Phase 1; a minimal supply
contest and recovery rules also enter the first multiplayer test. Full planets,
citadels, distributed seasonal play, and elaborate events remain staged later.
Reason: the first playable version must test the proposed social strategy game.

Additive: playtest plan, architecture/data extensions, configuration register,
release gates, rollback guidance, and copyable commit notes.

Breaking impact: no runtime, API, schema, or dependency changes are made by this
documentation revision. The earlier phase ordering is superseded by Section 3.
The season/reset design describes intended future behavior and must be disclosed
before players begin a competitive season.

Issue references: local requirements SC-R01 through SC-R08 only.
Commit hash: not available; no repository commit was made.
Release artifact: Sovereign_Conquest_Roadmap.txt, document version 00.01.00.

## 12. VALIDATION STATUS AND DOCUMENT ROLLBACK

Current repository integration (00.02.01): this revision reformats and integrates
the supplied roadmap. Its packet register records planned work, not completed
application behavior. The historical application validation in Section 18 is
preserved with its original assessment commit and limitations; it is not a claim
that those commands were rerun for this documentation integration.

Integration checks performed on 2026-09-27:

* Go 1.26.6: `go test -mod=vendor -count=1 ./...` passed from `server/`.
* All twenty numbered sections, eight requirements, eleven findings, and fifteen
  implementation packets are present; all fifteen tracker entries are Planned.
* All substantive source lines from Section 1 through Appendix A remain in order
  after accounting for Markdown formatting; the original appendix matches exactly.
* Thirty-two relative documentation links and anchors resolve across this roadmap,
  the README, and the changelog; whitespace checks pass.
* Application VERSION remains 01.06.03. The integration changes documentation only.

Container, database, security, and performance checks were not repeated for this
documentation integration. Section 18 records the earlier application assessment
and its limitations. Required checks for future runtime changes remain in Section 10.

To roll back this integration, revert its documentation commit through the normal
repository review process, including the associated README and changelog links.
For a later roadmap correction, restore the prior committed docs/ROADMAP.md
revision. Neither operation requires an application image rollback, database
restore or change to application VERSION. Keep the integration commit available
in Git history; the pre-integration 00.02.00 source artifact and supplied Appendix A
remain the provenance for this revision.

Historical validation and rollback note from document 00.02.00:

This revision evaluates the current application without changing its source.
Section 18 records the full baseline checks and limits. Automated source checks,
a fresh-cache API build, scanners and documentation checks were performed;
Docker/database integration, container scanning and local CodeQL were unavailable.
Do not interpret passing source tests as acceptance of the proposed gameplay.

The earlier 00.01.00 design is retained through the existing document's version
history. Appendix A preserves the originally supplied unversioned text. To undo
this assessment revision, restore document 00.01.00; to return to the original
brief, use Appendix A. Neither operation rolls back application code or data.
Future application releases require the migration and rollback plan in Sections
10 and 17, including the previously tested image digest and database recovery.

## 13. COPYABLE COMMIT NOTES

Current repository integration commit notes (00.02.01):

```text
docs(roadmap): integrate evaluated implementation plan 00.02.01

Purpose:
Keep the accepted product design and evaluated implementation plan in the source
repository as one canonical roadmap.

Changes:
* Import the complete roadmap as docs/ROADMAP.md with navigable Markdown headings.
* Add one status/dependency register for SC-I01-SC-I15, initially Planned.
* Preserve all twenty sections, SC-R01-SC-R08, F01-F11 and the original appendix.
* Preserve source assessment 03625c68a141f649aa87480b13d61527febceff1 and its
  historical validation evidence without claiming a new application assessment.
* Increment the document version to 00.02.01 for integration and traceability.
* Update documentation scope, changelog, current commit notes and rollback.

Validation:
Go 1.26.6: go test -mod=vendor -count=1 ./... passed from server/.
Content preservation, all twenty sections, eight requirements, eleven findings,
fifteen planned packets, the exact appendix, thirty-two relative documentation
links/anchors, and git diff --check passed. Section 18 retains historical checks
for 01.06.03; importing those results does not complete any work packet.

Compatibility:
Documentation only. Application version 01.06.03, runtime behavior, dependencies,
schema and configuration remain unchanged. Proposed milestone versions remain
reservations, not published releases.

Rollback:
Revert the documentation integration commit and its associated documentation
links. No application rollback or data restore is required.

References:
Document 00.02.00 source artifact; assessment commit
03625c68a141f649aa87480b13d61527febceff1; SC-R01-SC-R08; F01-F11; SC-I01-SC-I15.
Use actual review and commit references from Git history when available.
```

Historical copyable commit notes from document 00.02.00 follow. Their statements
about checks, commit creation and rollback describe the earlier assessment only:

```text
docs(roadmap): map 01.06.03 codebase to implementation plan 00.02.00

Purpose:
Turn the accepted game design into reviewable work on the existing Go,
PostgreSQL and browser application, starting with correctness and safety gaps.

Changes:
* Record baseline main commit 03625c68a141f649aa87480b13d61527febceff1.
* Map SC-R01-SC-R08 to existing modules and remaining behavior.
* Document economy, progression, session, reset, scheduling and UI findings.
* Add SC-I01-SC-I15 work packets, dependencies, source paths and acceptance tests.
* Define migrations, compatibility, feature rollout, release promotion and rollback.
* Record measured coverage, executed validation, scan caveats and remaining gates.
* Preserve the accepted design and original source appendix.

Validation:
Existing Go tests/race/vet/format/module checks, clean-cache API build, JavaScript
and shell syntax, gosec and govulncheck ran against the unchanged application.
Section 18 contains results and unavailable checks. Document coverage and
consistency review completed before saving.

Compatibility and risk:
Documentation only. Future code and schema changes remain proposed; the roadmap
identifies migration and rollback gates rather than claiming implementation.

Rollback:
Restore the prior 00.01.00 roadmap revision. No application rollback is needed.

References:
Application baseline 03625c68a141f649aa87480b13d61527febceff1; local requirements
SC-R01-SC-R08; proposed packets SC-I01-SC-I15. No new application commit or tag
was created. Add the actual documentation commit hash when committed.

```

## 14. SUPPORTING DESIGN REFERENCES

Ryan, Rigby, and Przybylski (2006), The Motivational Pull of Video Games:
A Self-Determination Theory Approach. Research context for autonomy, competence,
social connection, enjoyment, and future play; specific mechanics here still
require game-specific testing.
https://selfdeterminationtheory.org/SDT/documents/2006_RyanRigbyPrzybylski_MandE.pdf

CCP Fozzie / EVE Online (2013), Resource Shakeup in Odyssey: Just don't call it a
Cataclysm. Historical developer discussion of relationships among industry,
resource gathering, protection, and destruction; not a claim about current EVE
balance rules or evidence that these mechanics guarantee retention.
https://www.eveonline.com/news/view/resource-shakeup-blog

## 15. CURRENT CODEBASE ASSESSMENT

Historical assessment from document 00.02.00 follows. The word "current" in
this section refers to the source snapshot reviewed on 2026-09-27, not an
assertion that the repository integration commit was reassessed.

Reviewed on: 2026-09-27
Repository: https://github.com/paulkakell/sovereign-conquest
Branch: main
Reviewed commit: 03625c68a141f649aa87480b13d61527febceff1
Commit date: 2026-08-14
Application baseline: 01.06.03
Roadmap revision containing this assessment: 00.02.00

All current-code observations in Sections 15-20 refer to that exact commit.
The repository was cloned and read locally. GitHub's main branch reference
matched the clone. Changes on other branches are not part of this baseline.
No application code was changed, committed, deployed, or published in this review.

Source permalink root:
https://github.com/paulkakell/sovereign-conquest/tree/03625c68a141f649aa87480b13d61527febceff1

Path convention: paths below are repository-relative. G/ means
server/internal/game/, A/ means server/internal/api/, S/ means
server/internal/schema/, and W/ means web/static/. Function names and source
line anchors are evidence for this snapshot, not permanent future line numbers.
Proposed files and tables are explicitly marked new.

### 15.1. Architecture decision

Retain the Go service, PostgreSQL database, plain JavaScript client, and combined
API/web image. The game already implements much of the original roadmap.
Refine existing modules and add focused services around ExecuteCommand; a
framework rewrite or microservice split is not a prerequisite.

Current Go module: sovereignconquest. go.mod declares Go 1.25.0; CI and Docker
builders use Go 1.26.6. Dependencies are vendored. Current state-changing actions
use PostgreSQL transactions with a locked player row; trading also locks its
port. Turn regeneration is computed on state reads and commands, not a dedicated
turn worker. Preserve that on-demand behavior while fixing capacity policy.

The current browser uses bearer JWTs stored in localStorage. HttpOnly cookies in
the earlier design are a possible later transport migration, not an implemented
capability or a prerequisite for the gameplay plan. Session revocation is an
immediate prerequisite regardless of transport.

### 15.2. Existing capability versus remaining work

SC-R01 / Commerce and supply
Already present: three commodities, stock-sensitive prices, event price effects,
atomic trades, planet production, and equipment consumption by mine deployment.
Evidence: G/trade.go:29, G/economy.go:5, G/tick.go:17, G/mine.go:37.
Remaining: repair BUY demand, add economic journals, actual upkeep/consumption,
bounded freight rewards, and deliveries that sustain a contested objective.

SC-R02 / Useful roles and cooperation
Already present: SCOUT, TRADER, FREIGHTER, INTERCEPTOR; corporation creation,
open joining, leaving, chat, deposits, and leadership-only withdrawals.
Evidence: G/shipyard.go:23, G/corp.go:22,263.
Remaining: meaningful recon/hauling/protection tradeoffs, controlled recruitment,
explicit asset permissions, contribution records, intel sharing, escrow and
asynchronous escorts. Extend these ship names; do not rename saved ship types.

SC-R03 / Strategy and intelligence
Already present: per-player discovery, port scan timestamps, scan-age display,
freshness-weighted route suggestions restricted to discovered geography, mines.
Evidence: G/intel.go:32,109; G/market.go:12; G/route.go:35,81,259,280.
Remaining: ordinary-player map, source/visibility-aware shared observations,
feasible route alternatives, previews, encounters, and ship/fighter combat.
Current combat consequences are mine/raider credit penalties. There is no ship
condition or destruction system and no ATTACK command in the engine switch.

SC-R04 / First session and goals
Already present: registration, welcome log, status/sector/port panels, terminal,
direct messages and bug/abuse reporting.
Evidence: A/server.go:238-304; W/index.html:88-132; W/app.js:254-397.
Remaining: guided freight, persisted tutorial progress, structured action forms,
next goals, player map, meaningful first upgrade, and relevant return briefing.

SC-R05 / Session pacing and recovery
Already present: regeneration under a player lock; 120-second default; protected
space prevents mine deployment. Evidence: G/engine.go:20; G/mine.go:44;
server/internal/config/config.go:39.
Remaining: full-day storage policy, docking/encounter exposure, loss/recovery,
attack windows, and finite escort commitments. The starter has 100 turns of
capacity; the highest base hull has 140. Even the highest upgraded cap of 240
stores only eight hours at the current regeneration rate.

SC-R06 / Fairness and late entry
Already present: Protectorate sectors and shipyards, cargo/balance validation,
limits on deployed mines. Evidence: G/protectorate.go, G/mine.go, G/trade.go.
Remaining: reliable protected spawn, bounded starter support, expansion upkeep,
anti-farming reward rules, shared-asset authority, and contribution-based value.

SC-R07 / Seasons and history
Already present: season records, player season references, credit leaderboard,
operator soft wipe, player ranks, and planet names.
Evidence: S/schema.go:130-153; G/rankings.go; G/admin.go.
Remaining: announced end, objective scoring, complete reset manifest, settlement,
archives, medals, ship identity, and lasting records of meaningful contributions.

SC-R08 / A populated, persistent universe
Already present: seeded graph generated once; default 200 sectors; connected
ring with extra connections; price events and raider penalties.
Evidence: G/universe.go:18; G/events.go; G/event_leader.go.
Remaining: reliable generation completion, compact pilot configuration, freight
activity, persistent world metadata, and explicit frontier expansion. Changing
UNIVERSE_SECTORS does not expand a populated world; EnsureUniverse exits when
any sector already exists.

### 15.3. Findings that determine implementation order

#### F01. BUY demand can permanently saturate. Confirmed from source.

G/tick.go:19-24 increments every commodity stock regardless of mode. Selling to
a BUY port increments the same stock and rejects orders beyond capacity
(G/trade.go:123-156). BUY stock has no normal draining path. Fix consumption
for BUY inventory and replenishment for SELL inventory before new trade rewards.
Priority: first correctness patch. Acceptance: a saturated buyer recovers demand
after a due tick; stocks stay bounded and trades remain atomic.

#### F02. Free informational commands award repeatable XP. Confirmed from source.

G/engine.go:289-299 awards successful-command XP; commandCost at 374-404 makes
HELP, MARKET, ROUTE, rankings, and several corporation actions free.
G/ranks.go:267-329 rewards them. Credit deposit/withdrawal cycles also earn XP.
Current ranks appear cosmetic, so this is reward-integrity failure, not evidence
of existing combat-stat advantage. Award progress for verified useful outcomes;
remove awards for reads and reversible transfers before adding medals or scores.

#### F03. Soft wipe retains fleet value and deletes history. Confirmed from source.

G/admin.go:46-58 resets credit/cargo/turn capacities but leaves ship_type, ship
upgrade counters, XP and level. A retained FREIGHTER still has a 42,000-credit
resale value through G/shipyard.go:151-171. The reset also deletes logs/intel/events
and does not reset port stock or archive standings. Do not use this endpoint as
the production competitive-season transition without the M6 work below.

#### F04. Password changes do not revoke existing sessions. Confirmed from source.

Claims.SessionVersion exists in server/internal/auth/auth.go:13-17, but login
and registration mint version-zero tokens with seven-day lifetimes
(A/server.go:311,350). authMiddleware does not check a persisted session version;
handleChangePassword only updates the hash and timestamp (398-402). Add and
enforce session_version for normal and admin routes, including bootstrap recovery.
Also align the 8-100-byte API password limit with bcrypt's 72-byte maximum.

#### F05. Scheduled-job locks do not deduplicate intervals. Source-derived risk.

G/ticker_lock.go:23-43 releases its transaction lock after each invocation.
Staggered API processes can each apply production within the same interval.
G/event_leader.go also locks per invocation; G/protectorate.go:184-190 has no
such lock and ignores errors. Add durable due-time/cursor state before supply,
contract expiration, or score settlement relies on scheduled work. A multi-worker
database regression is required; this review did not run that scenario.

#### F06. Shared-asset ownership and membership policy need correction/definition.

G/corp.go:112-143 currently allows open joining. That is existing product behavior,
not an authentication bypass. G/planet.go:123-127 can set both player and corp
owners, and canAccessPlanet at 391-398 accepts either. A founding player can
retain access after leaving. Adopt explicit personal-versus-corporate authority
with a recorded migration, role checks, and configurable recruitment before
valuable shared deliveries. Existing withdrawals already require LEADER/OFFICER
and a conditional sufficient-balance update; preserve those protections.

#### F07. Corporation chat fan-out has a database interaction risk.

G/corp.go:214-225 runs tx.Exec while memberRows is still being consumed and ignores
the error. pgx may report a busy connection; the message can be stored without
all member log deliveries. Reproduce with PostgreSQL, replace with checked
INSERT...SELECT or close rows before inserting, and assert every recipient.
Also propagate required save/log/commit errors currently ignored in
G/engine.go:306-310,334-338 and reset paths. Do not claim a live incident was observed.

#### F08. Protected, useful starting conditions are not guaranteed.

A/server.go:240 selects the lowest sector ID, while Protectorate selection is
separate. Raider entry damage does not consistently consult a shared protection
rule. G/universe.go returns when any sector exists despite staged nontransactional
generation. Add a protected spawn selector and generation completion checks;
test partial startup failure and restart before relying on a tutorial route.

#### F09. Schema and accounting support are insufficient for escrow.

S/preflight.go and S/schema.go run monolithic startup DDL without ordered migration
history. CommandRequest has no idempotency key, and text logs are not a transaction
journal. This does not negate the existing transaction locks: it means successful
requests retried after lost responses have no deduplication guarantee. Establish
migrations, receipts, and journals before funding contracts.

#### F10. Browser compatibility code is functional but obstructs new actions.

W/compat-010601.js repairs existing command payloads, log/event shapes, token keys,
password-change flow and attachment authentication. Do not remove it blindly.
Its correctedCommandPayload function also reads the terminal DOM and overrides
submitted type/action. New structured buttons could therefore execute a stale
terminal command. Replace these adaptations with explicit shared modules before
introducing new action forms; cover each existing adaptation with browser tests.

#### F11. Production and release configuration need a defined safe path.

docker-compose.yml defaults to development and hardcodes sslmode=disable. The
production runtime rejects that database mode. Provide a tested TLS deployment
override rather than weakening validation. The publisher pushes main, numeric
version and SHA image tags before scan/smoke completion, independently of the CI
workflow, and republishes them on a schedule. git ls-remote --tags origin returned
no Git tags during review. Introduce validated candidate promotion, immutable
numeric releases, matching Git tags, and recorded rollback digests.

## 16. DEPENDENCY-ORDERED IMPLEMENTATION PLAN

This is the execution order for the existing 01.06.03 codebase. It takes priority
over the earlier greenfield phase grouping in Section 3. Keep completed features
available while extending them. The application versions below are proposed
release reservations, not created tags, commits, or published builds. Rebase the
numbers if main changes before implementation.

M1 / 01.06.04 / Correctness and account safety
M2 / 01.07.00 / Durable actions, scheduling, and configuration
M3 / 01.08.00 / Player interface, intelligence, and corporation authority
M4 / 01.09.00 / Freight, escrow, onboarding, and return goals
M5 / 01.10.00 / Protected conflict, escorts, and the first relay contest
M6 / 01.11.00 / Competitive seasons, archives, and contribution recognition
M7 / 01.12.00 / Population-led expansion and deeper planetary supply

Each milestone is a separately reviewable release. During development, identify
PR artifacts by the intended version plus commit SHA; publish the numeric release
tag only once its full gate passes. The work packet labels below are proposed
backlog references, not GitHub issues already created.

### M1. CORRECTNESS AND ACCOUNT SAFETY

#### SC-I01. Establish a database-backed baseline and ordered migrations.

Depends on: current main only. Addresses F03-F09 and the Section 10 release gates.
Existing touchpoints: S/preflight.go, S/schema.go, server/cmd/api/main.go,
.github/workflows/ci.yml, .github/workflows/build-validation.yml,
scripts/release-smoke.sh.
New: server/internal/testdb/, S/migrations/, schema migration runner and
schema_migrations table; optional server/cmd/migrate entrypoint.
Work: seed reusable fresh and 01.06.03 database fixtures; establish a checksummed
baseline under a migration lock; test fresh/upgrade catalog parity. Preserve the
legacy preflight until old-schema upgrade fixtures pass. Do not simply reorder
or remove startup DDL and assume existing deployments will upgrade.
Acceptance: fresh install, restored current fixture, and legacy missing-column
fixture reach the same supported schema; rerun is a no-op; concurrent migrators
serialize; drift fails visibly. CI provisions disposable PostgreSQL for behavioral
tests. Retain the existing smoke test but require successful SCAN with expected
state change, not merely the presence of a message.

#### SC-I02. Repair the economy, progression, chat, and rejection paths.

Depends on: SC-I01 test harness. Addresses F01, F02, F07.
Existing touchpoints: G/tick.go, trade.go, economy.go, ranks.go, engine.go,
corp.go and their tests; server/internal/rules/ as appropriate.
Work: consume BUY stock, replenish SELL stock, suppress read/transfer-cycle XP,
and remove or uniquely bound corporation creation/membership rewards so free
CREATE/LEAVE and JOIN/LEAVE loops cannot substitute for the original exploit.
Make required transaction writes checked and repair chat fan-out. Keep current
order-pricing semantics initially; document whole-order quote behavior rather
than silently replacing it with marginal pricing.
Acceptance: saturated BUY demand recovers; SELL stock replenishes; repeated HELP,
MARKET, corp INFO, deposit/withdraw, CREATE/LEAVE and JOIN/LEAVE cycles produce
no repeatable progression; successful chat reaches current members only; failed commands do not persist partial trades;
parallel trades never oversell inventory or create negative balances.

#### SC-I03. Enforce account revocation and consistent reset/spawn state.

Depends on: SC-I01. Addresses F03, F04, F08.
Existing touchpoints: server/internal/auth/auth.go, A/server.go, A/admin_guard.go,
G/initial_admin.go, G/admin.go, G/shipyard.go, G/protectorate.go, G/universe.go.
New: users.session_version; a shared starter-state constructor and protected
spawn rule; minimal reset audit manifest before the later full season system.
Work: persist/mint/validate session versions, increment on credential recovery,
enforce required password change for administrative mutations, and document the
72-byte password limit. Make registration and reset use consistent hull/upgrades/
capacities and a valid protected sector. Correct retained resale value; disable
competitive season use of legacy soft wipe until full archive/settlement exists.
Acceptance: old token rejected after password change/recovery, fresh token works;
nonadmin and password-required admin cannot reset; reset pilot cannot sell a
retained freighter; interrupted generation is detectable and recoverable; starter
route remains accessible under seeded events.

#### SC-I04. Make release promotion and production configuration trustworthy.

Depends on: the preceding M1 checks. Addresses F11.
Existing touchpoints: Dockerfile, server/Dockerfile, docker-compose.yml,
.env.example, server/cmd/api/runtime_validation.go, server/scripts/build_api.sh,
.github/workflows/ci.yml, build-validation.yml, publish-ghcr.yml,
server/internal/build/workflow_test.go, docs/CONFIGURATION.md.
Work: build a candidate, run all required tests/scan/smoke against its digest,
then promote the tested digest to an immutable version and moving main tag.
Create the matching Git release tag at the verified source commit. Scheduled
rebuilds use distinct candidate identities and do not overwrite numeric releases.
Add an explicit production database URL/TLS override and CA handling. Remove
automatic checksum-verification disabling from the nonvendor build fallback;
retain vendored builds and deliberate private-module configuration.
Acceptance: failed validation cannot promote; release version/SHA/digest agree;
production TLS connection succeeds and plaintext mode still fails; prior digest
and restore instructions are recorded. Update workflow-contract tests to check
required behavior without obstructing intentional dependency upgrades.

### M2. DURABLE ACTIONS, SCHEDULING, AND CONFIGURATION

#### SC-I05. Add command receipts, economic journals, and typed events.

Depends on: M1. Enables SC-R01, SC-R02, SC-R07.
Existing touchpoints: A/server.go handleCommand, G/types.go, engine.go, store.go,
trade.go, corp.go, mine.go, planet.go, shipyard.go, and S/migrations/.
New: command_receipts, economic_entries, domain_events, optional event_outbox;
focused receipt/ledger/event helpers under G/.
Work: accept a request_id with a normalized request hash, scoped to actor and
season; atomically persist the result with mutations. Same key/body returns the
committed result; same key/different body returns a conflict. Introduce journal
opening balances for existing assets, not invented historical transactions.
Classify faucets, sinks, and transfers. Add stable event IDs, season, actor,
object, visibility, timestamp and structured payload; preserve legacy text logs.
Acceptance: lost-response retries change balance/cargo/turns once; parallel
duplicates return one outcome; state and journals reconcile; unauthorized actor
IDs fail; published events never describe a rolled-back mutation.
Compatibility: legacy commands can omit request_id during a documented migration
window. New escrow, settlement and combat commands require it from first release.

#### SC-I06. Persist scheduled work and centralize balance rules.

Depends on: SC-I05 and ordered migrations. Addresses F05 and SC-R05.
Existing touchpoints: G/ticker_lock.go, tick.go, event_leader.go, protectorate.go,
engine.go RegenTurns, shipyard.go, admin.go, initial_admin.go,
server/internal/config/config.go and docs/CONFIGURATION.md.
New: scheduled_jobs with due cursor/interval, rules version record, configuration
for base turn storage and bounded catch-up. Preserve per-player lazy regen.
Work: under a job lock, check database time, apply due work and advance the cursor
in one transaction. Record random outcomes per bucket so retries do not reroll.
Use the same mechanism for all recurring workers; surface lag and failures.
Prototype a 720-turn base at 120 seconds, document how existing hull/upgrade
bonuses are rebased, and preserve earned upgrades without silently deleting them.
Acceptance: one and two staggered workers yield equal production; restart/retry
cannot duplicate an interval; downtime catch-up is bounded; all registration,
shipyard and reset paths agree on capacities; a full day's budget can support
useful sessions without hundreds of mandatory repetitive actions.

### M3. PLAYER INTERFACE, INTELLIGENCE, AND CORPORATION AUTHORITY

#### SC-I07. Consolidate the browser contract, then add structured actions and map.

Depends on: SC-I05; can develop in parallel with SC-I06.
Existing touchpoints: W/app.js, compat-010601.js, bug.js, index.html, style.css;
A/server.go, G/types.go, store.go, route.go; A/ui_test.go and server/internal/build/web_contract_test.go.
New: W/api.js, commands.js, session.js and focused views; player-scoped map/read
handlers and command preview service. These are proposed modules, not a rewrite.
Work: move all compatibility behaviors into explicit functions used by terminal
and buttons, including token key migration, logs/events, password flow and
authenticated downloads. Stop deriving payload type/action from unrelated DOM
input. Add move/trade/loadout forms, typed previews, known-risk and cost displays,
and an ordinary-player map using authorized discoveries only.
Acceptance: every old terminal command still round-trips correctly; stale input
cannot alter a button command; password/attachment/message flows still work;
keyboard and narrow-screen controls are usable; map never returns the admin
universe view. Preview is advisory and execution revalidates costs/state; stale
quotes fail clearly or require renewed confirmation before changed commitments.
Tests: add browser behavior coverage for all shim adaptations before removing
the shim; include direct API authorization tests independent of UI visibility.

#### SC-I08. Extend intel, practical loadouts, and corporation permissions.

Depends on: SC-I05 and SC-I06; UI work depends on SC-I07. Enables SC-R02, SC-R03,
and SC-R06. Centralized capacity rules precede ship capability backfills.
Existing touchpoints: G/intel.go, market.go, route.go, shipyard.go, types.go,
corp.go, planet.go; S/migrations/; structured client views.
New: intel_observations/intel_shares, corp_invites/role policy/audit, explicit
planet ownership constraint and founding-player history where needed.
Work: retain scan timestamps and add source, season and sharing scope; extend
routes to current free cargo, credits, known hazards, and several useful choices.
Keep SCOUT, TRADER/FREIGHTER and INTERCEPTOR saved identifiers while adding
recon, hauling and later escort capability. Preview switching costs.
Add open/application/invite recruitment settings and leader-controlled roles.
Recheck permissions under the same transaction as treasury/storage changes.
Migrate dual-owned planets to declared corporate or personal ownership, retaining
the founder in history; unowned planets remain valid. Produce a mapping report
before changing existing titles. Preserve current treasury withdrawal restrictions.
Acceptance: former members lose future restricted reads/writes; member cannot
self-promote or spend restricted assets; leadership changes serialize; shared
intel never gains freshness when copied; route plans fit actual resources;
existing ships and balances survive backfill. Previously viewed knowledge cannot
be erased from a human player's memory; define server access revocation precisely.

### M4. FREIGHT, ESCROW, ONBOARDING, AND RETURN GOALS

#### SC-I09. Implement one freight state machine for NPC and player contracts.

Depends on: SC-I05, SC-I06, SC-I08. Enables SC-R01, SC-R02, SC-R04, SC-R08.
Existing touchpoints: G/engine.go, types.go, trade.go, corp.go, store.go;
A/server.go and the new structured client modules.
New: G/contracts.go, freight.go; contracts, escrow accounts/entries, deliveries,
contract_events, npc_reward_budgets and due-job records.
Work: implement offered -> accepted -> delivered/failed/expired/cancelled with
immutable destination, goods, reward and deadline after acceptance. Reserve
issuer funds at posting, reserve appropriate cargo/commitments at acceptance,
and settle delivery, cancellation and expiry atomically. Reuse the journal and
receipts. Allow only authorized corporation funding. Decide disband/leave/death
behavior in the contract terms. NPC jobs use bounded server budgets.
Acceptance: concurrent acceptance has one winner; delivery versus expiry has one
outcome; payout/refund occurs once; insufficient funds prevent posting; goods
cannot be delivered twice; disconnect does not strand escrow. Test deliberate
friendly-account cycles and duplicate requests before recognizing contributions.

#### SC-I10. Deliver the first expedition and a useful return screen.

Depends on: SC-I07, SC-I09. Enables SC-R04, SC-R05 and the first real user pilot.
Existing touchpoints: A/server.go registration/state; G/store.go, events.go,
protectorate.go; W/index.html/app.js replacements and views.
New: G/onboarding.go, briefing.go; onboarding_progress, player_goals,
player_activity_cursor; authorized /api/briefing and job/goal views.
Work: generate a viable protected freight opportunity using the contract service,
two understandable route choices, an affordable capability upgrade, and a next
goal. Resume progress across logins. Briefing returns relevant changes after a
stable event cursor, current commitments and suggested next actions. Preserve
both a five-minute contribution and a twenty-minute expedition.
Acceptance: a new player completes delivery, understands profit/cost, chooses an
upgrade and returns to the next goal without coaching; repeated tutorial claims
do not duplicate rewards; no profitable starter-account resource transfer loop;
quiet-world jobs remain useful but bounded. Browser tests cover refresh, expired
job, insufficient cargo/turns, and unread/relevant-event behavior.

### M5. PROTECTED CONFLICT, ESCORTS, AND ONE RELAY CONTEST

#### SC-I11. Implement exposure, encounters, retreat, and recovery before losses.

Depends on: SC-I05-SC-I10. Enables SC-R03, SC-R05, SC-R06.
Existing touchpoints: G/engine.go movement, mine.go, events.go, protectorate.go,
shipyard.go, types.go; previews and command dispatch.
New: G/docking.go, encounters.go, combat.go, fighters.go, recovery.go;
ship-condition state, fighter inventory and owned sector deployments, encounters,
docking/exposure state, replacement grants and attack windows.
Work: define protected docking, cargo exposure, committed encounters, retreat,
bounded losses, and basic replacement/earning access. Apply one protection policy
to mines, raiders, missions and attacks. Resolve resource-based outcomes with
readable reports; do not add reflex timing. Add explicit ATTACK and fighter
DEPLOY/RETRIEVE actions with validated targets, ownership, costs, and bounded
ship-versus-sector-defense resolution. Journal fighter deployment, depletion,
retrieval and destruction; define protected-space restrictions before release.
Keep new hostile mechanics disabled until the rules and recovery gate pass.
Acceptance: simultaneous dock/move/attack resolves once; docking cannot cancel an
already committed encounter; defender can be offline; announced windows hold;
replacement aid cannot create sellable profit; loss reports show useful cause
and counterplay. Concurrent attacks/retrieval cannot spend the same fighters
twice, players cannot retrieve another owner's defenses, and protected-sector
rules apply to deployment and attacks. Confirm safe start and affordable recovery
with real pilots.

#### SC-I12. Add bounded escorts and relay supply/control.

Depends on: SC-I09 and SC-I11; never ship exposed relay competition first.
Existing touchpoints: shipCatalog and new loadouts, contract service, journal,
scheduled jobs, previews, corporation permissions and player map.
New: G/escort.go, relay.go; escort_assignments, relays, relay_supply,
relay_control, relay_windows and verified objective_contributions.
Work: bind finite escort capability, turns and assets to one accepted mission
until completion/expiry under explicit cancellation rules. Implement one relay
type with bounded demand: organics sustain operation, ore repairs damage and
equipment supports defense. Delivered goods meet actual deficits. Add blockade
and interception objectives, supplied control, upkeep and advertised vulnerability
windows. Use existing planet output as one supply source before adding complex
industrial recipes.
Acceptance: scout, hauler and interceptor each make useful contributions across
different login times; escort cannot protect two commitments with the same
assets; a successful delivery changes relay state; overdelivery and friendly
cycling do not manufacture contribution rewards; small groups can interrupt a
shipment without conquering an empire; no duplicate combat/delivery resolution.

### M6. COMPETITIVE SEASONS AND LASTING HISTORY

#### SC-I13. Replace soft wipe with an audited season transition.

Depends on: SC-I05, SC-I06, SC-I12. Enables SC-R07.
Existing touchpoints: G/admin.go, rankings.go, ranks.go, universe.go,
A/admin_guard.go and admin endpoints; S/migrations/.
New: seasons rule/end fields, standings snapshots, season_transition records,
world identity, archives, achievements and named-asset history.
Work: implement prepare -> freeze -> settle -> archive -> reset -> open with a
transition ID, admission control and checkpoints. Settle/cancel contracts by
published terms, archive meaningful events, and reset an explicit manifest:
credits, fleet type, upgrades, XP policy, cargo, turns, ports, planets, mines,
events, intel, objectives, outstanding commitments and scheduled jobs. Preserve
only published identity/history/cosmetic benefits. Keep one active season.
Publish relay scoring, finish time and tie rules before participation. Expand
the one-relay M5 prototype to several strategically separated scoring relays,
with a declared layout and supply/defense commitments that one concentrated force
cannot automatically satisfy everywhere. Before scoring opens, adopt a bounded
planetary production/holdings policy or minimum expansion upkeep; test its effect
on late entry and wealth concentration. Deeper planetary simulation remains M7.
Start with the proposed eight-week duration as a tunable production hypothesis.
Acceptance: transition retry does not create another season; injected failure
does not strand funds or partially reset players; no retained ship resale
advantage; current commands/workers cannot write across the closing boundary;
archived standings agree with verified event totals; distributed objectives
remain contestable and late arrivals have useful, attainable contributions under
the bounded production/holdings rules. Rehearse backup/restore
and state the cutoff for any post-reset actions that recovery could lose.

#### SC-I14. Run the complete multiplayer pilot and tune from evidence.

Depends on: M5 for a short noncompetitive pilot; M6 for a scored season trial.
Use one compact world and approximately 12-20 participants with mixed schedules.
Measure the Section 4 outcomes: next goals, useful roles, returns, recovery,
concentration, available demand, resource flows and absence costs. Check five-
minute visits and twenty-minute expeditions without requiring everyone online.
Technical acceptance: no unresolved balance/escrow/score reconciliation errors,
unauthorized intel, interval duplication, or loss outside published rules.
Product acceptance: participants form specific plans and retain plausible ways
to contribute after defeat or late arrival. The sample informs design; it does
not establish statistical retention targets or prove long-term commercial demand.

### M7. POPULATION-LED EXPANSION AND PLANETARY SUPPLY

#### SC-I15. Expand only after the complete pilot supports it.

Depends on: SC-I13 and SC-I14. Completes SC-R01, SC-R06, SC-R08 depth.
Existing touchpoints: G/universe.go, planet.go, tick.go, events.go, route.go,
load/ownership/graph queries, and season archives.
New: world_expansions with seed/generator/rule version and completion status;
planet operating-input/upkeep rules; further discovery/job templates.
Work: append frontier sectors atomically while preserving existing IDs/edges,
ownership and intel. Ensure new routes join the reachable graph. Connect colony
operation and citadel defenses to transparent supply costs and distributed
defensive commitments. Add limited exploration/events when existing activity
and density justify them; keep one public universe initially.
Acceptance: interruption and retry add each frontier once; old routes remain
valid; no player becomes stranded; larger holdings impose visible supply costs;
small-world and larger-world load tests meet recorded targets. Extra commodities,
many server variants and elaborate invasions remain deferred unless pilot
evidence identifies a specific benefit.

## 17. MIGRATION, COMPATIBILITY, AND ROLLOUT CONTRACT

Proposed migration families, executed in the milestone order above:

* M1: baseline/checksums, session_version, reset consistency and audit support.
* M2: command receipts, economic journal, typed events, durable jobs, rules version.
* M3: ownership normalization, recruitment/role policy, immutable intel sharing,
  compatible ship capabilities and player map/read models.
* M4: contracts/escrow/delivery, NPC budgets, tutorial progress, goals and cursors.
* M5: docking/encounters/recovery, escort commitments, relays and supply/control.
* M6: season transition, standings, archives, achievements, world identity.
* M7: expansion history and deeper operating inputs/upkeep.

Use expand -> backfill -> verify -> switch. Drop obsolete columns or compatibility
paths only in a later declared release. Current player IDs remain stable; do not
accidentally introduce a second player per account while adding historical
results. New competitive records carry season_id and, where needed, world_id.
Preserve globally stable sector identifiers, or explicitly migrate every foreign
key before any change to their scope. Old scan data becomes a recorded opening
snapshot with honest provenance; do not fabricate historical observations.

Preflight actual data before adding uniqueness and balance constraints. Check
active-season count, duplicate player/user relationships, negative balances,
invalid roles, dual planet ownership, and inconsistent ship stats. Report and
repair intentionally; never silently discard player assets to satisfy a check.
Allow unowned planets while disallowing simultaneous personal and corporate title
once the ownership migration is complete.

Existing configuration to preserve and document:
DATABASE_URL, JWT_SECRET, ADMIN_SECRET, INITIAL_ADMIN_USERNAME,
INITIAL_ADMIN_PASSWORD, UNIVERSE_SEED, UNIVERSE_SECTORS, TURN_REGEN_SECONDS,
PORT_TICK_SECONDS, PLANET_TICK_SECONDS, EVENT_TICK_SECONDS,
PROTECTORATE_TICK_SECONDS, HTTP_ADDR, WEB_ROOT, APP_ENV, TRUST_PROXY_HEADERS,
SC_IMAGE and SC_PULL_POLICY. Preserve nonpositive ticker-disable behavior in
tests and clearly document which settings affect an existing world.

Proposed new configuration, not currently implemented:
TURN_STORAGE_HOURS (initial hypothesis 24), JOB_MAX_CATCHUP_INTERVALS,
CONTRACTS_ENABLED, INTEL_SHARING_ENABLED, RELAY_PVP_ENABLED, SEASON_MODE,
SEASON_DURATION_DAYS (hypothesis 56). Validate bounds and persist active rule
versions. Define job-specific catch-up limits and attack windows before enabling
their feature; do not invent silent production defaults.

When disabling a feature, stop new commitments while continuing safe settlement
of existing ones. Turning off contracts must not strand escrow. Turning off PvP
must not duplicate or erase already committed encounter outcomes. Migrate the
browser and API together while preserving old command fields through the stated
compatibility window. Return human-readable command errors plus stable codes.

Stage rollout: disposable database -> restored anonymized 01.06.03 fixture ->
private pilot universe -> release candidate -> validated promotion. Until durable
jobs are implemented, use one API process per universe as an interim constraint;
do not imply the current per-invocation locks provide replica-independent rates.

Rollback each release to a pinned prior image digest while additive schema remains
compatible. Disable new commitments first, settle or safely suspend outstanding
work, and reconcile journals. Incompatible season transitions require a verified
database restore or forward repair, not just an old container. Record backup ID,
schema version, source SHA, image digest, recovery time and acceptable data-loss
window. Retain original current-version fixtures and the previous roadmap version.

## 18. VALIDATION EVIDENCE AND REQUIRED NEW COVERAGE

The results below are preserved from the 00.02.00 assessment at commit
03625c68a141f649aa87480b13d61527febceff1. This integration does not promote
historical PASS, NOT RUN or GitHub-reported results into new validation evidence.
Required future coverage remains proposed work, not completed tests.

The following verification describes the reviewed baseline, not future features.
Local execution used verified official Go 1.26.6 to match CI and Node 24.19.0.
Commands ran against unchanged source at the baseline SHA. Run Go commands from
server/; JavaScript paths below are under web/static/ and shell paths are relative
to the repository root. Output/cache paths refer to disposable review files.

PASS: go test -json -count=1 -coverprofile=`<scratch>`/coverage.out ./...
30 top-level tests plus 10 subtests; seven packages contain tests.
Statement coverage: 9.6% overall, game 5.7%, API 16.9%, auth 73.1%, rules 95.5%.
The cmd/api, config, db, schema and util packages measured 0.0%.

PASS: go test -race -count=1 ./...

PASS: go vet ./...

PASS: gofmt -l . (no files listed; no formatting edits made).

PASS: go mod verify (module cache verification; this does not by itself compare
committed vendor source against upstream archives).

PASS: CGO_ENABLED=0 GOOS=linux GOCACHE=`<new-empty-cache>` go build -mod=vendor
-trimpath -buildvcs=false -o `<scratch>`/sovereign-api ./cmd/api.
This is a fresh-cache source binary build, not a container/image build.

PASS: node --check on app.js, compat-010601.js and bug.js.

PASS: bash -n scripts/release-smoke.sh; sh -n server/scripts/build_api.sh.

PASS: gosec v2.28.0, gosec -quiet ./..., no reported findings under existing
suppressions. This does not prove that no security defects exist.

PASS: govulncheck v1.7.0, govulncheck ./... and -show verbose ./...:
no reachable vulnerabilities and no affected imported packages.

Dependency qualification: the scan reported three advisories at required-module
level in unused x/crypto packages. GO-2026-6355 and GO-2026-6354 affect SSH code
and are fixed in x/crypto v0.56.0. GO-2026-5932 concerns unmaintained openpgp with
no fixed version. This app imports bcrypt rather than those affected packages.
Review and test the pending x/crypto update, regenerate vendor with module files,
and reassess before introducing new SSH/openpgp imports. Do not describe all
required modules as free of known advisories.

PASS: existing benchmark command: go test -run=^$ -bench=. -benchmem ./...
Only the request-limiter benchmark exists: 120.5 ns/op, zero allocations in this
run. It measures no database, trade, route, contract, or multiplayer workload.

NOT RUN LOCALLY: Docker/Compose builds/config rendering, fresh PostgreSQL smoke,
existing-save migration/restore exercises, browser behavioral/journey tests,
image scanning and CodeQL. No browser behavior suite exists in the baseline.
Docker and PostgreSQL were absent; attempted disposable database package setup was blocked
by the environment's setgroups/seteuid permissions. No live database was used.
No new behavioral tests were added because this task changes the roadmap only;
the required new tests are specified in each implementation packet.

Tracked application source remained unchanged after validation.


GitHub-reported status for the same reviewed commit:

* CI run 31804000115: success, 2026-08-14.
  https://github.com/paulkakell/sovereign-conquest/actions/runs/31804000115
* Build Validation run 31804000061: success, 2026-08-14.
  https://github.com/paulkakell/sovereign-conquest/actions/runs/31804000061
* Publish GHCR Image run 36287089110: success, 2026-09-27.
  https://github.com/paulkakell/sovereign-conquest/actions/runs/36287089110

These are observed workflow conclusions, not locally reproduced container tests.
The repository's current image smoke test checks health/version/static content,
registration, state and a loose SCAN response. It runs APP_ENV=development and
disables all periodic jobs, so it does not validate production TLS/configuration.
It does not exercise trading, corporate authority, chat fan-out, migration from
an existing database, seasonal reset, or multi-worker scheduling.

Required regression scenarios before M1-M6 are accepted:

* BUY/SELL stock evolution, conservation, concurrent trades and rejected actions.
* Free-command/transfer-cycle rewards, stale quote handling and duplicate requests.
* Token revocation, forced-password admin limits, corporation permissions,
  leadership changes and former-member access.
* Corp chat delivery and rollback on required log/journal failure.
* Fresh install, legacy upgrade, real 01.06.03 fixture upgrade and restore.
* Two staggered workers, retry after commit, process restart and bounded catch-up.
* Contract accept/deliver/cancel/expire races and zero stranded escrow.
* Intel authorization/provenance, player map privacy and route affordability.
* Onboarding reward replay, useful next goal and narrow-screen/keyboard flow.
* Dock/move/attack ordering, offline defense, finite escorts and replacement abuse.
* Relay deficits, supplied control, overdelivery, score replay and allied cycling.
* Complete reset manifests, fleet resale, concurrent registration, archives,
  repeated transition IDs and recovery after partial failure.

Use disposable PostgreSQL fixtures and browser tests for behavior, retaining pure
Go unit tests for pricing, routes, ranks and deterministic rules. Avoid treating
source-text assertions or syntax checks as substitutes for end-to-end gameplay.
Collect per-feature coverage for changed critical paths; do not set a global
percentage as a substitute for those scenarios.

Performance work: before M4-M6 release, benchmark same-port trade contention,
map/route reads, briefings, scheduled-job catch-up, delivery/escrow resolution and
large season transitions. Test the 12-20-player pilot load and a documented growth
scenario; record p95 latency, error rate, lock wait, rows scanned and job lag.
Set numerical latency targets before measuring against them; this review does
not claim an established production workload or target. Existing route search
performs repeated graph traversal and a BUY/SELL pair search, so profile it
before enlarging the universe rather than adding speculative caches.

Observability work: keep the existing structured HTTP request logs and probes.
Add correlated command/receipt/event IDs, ledger reconciliation checks, due-job
lag, escrow aging, denied resource access, and season-transition state. Avoid
logging secrets, tokens, hidden intel or private message content. Add post-release
checks that inspect outcomes and balances, not only whether the process is alive.

## 19. DEPENDENCIES, RELEASE TRACEABILITY, AND OPEN WORK

Open dependency PRs observed on 2026-09-27, outside the reviewed main commit:

* #21: pgx 5.10.0 -> 5.11.0.
* #20: x/crypto 0.55.0 -> 0.57.0.
* #18 and #19: root/server Go builder image 1.26.6 -> 1.27.1.
* #14: chi 5.3.1 -> 5.3.2.
* #13: attest-build-provenance v3 -> v4.
* #12: trivy-action 0.35.0 -> 0.36.0.
* #11: setup-buildx-action v3 -> v4.
PR links: https://github.com/paulkakell/sovereign-conquest/pull/21, https://github.com/paulkakell/sovereign-conquest/pull/20, https://github.com/paulkakell/sovereign-conquest/pull/18, https://github.com/paulkakell/sovereign-conquest/pull/19, https://github.com/paulkakell/sovereign-conquest/pull/14, https://github.com/paulkakell/sovereign-conquest/pull/13, https://github.com/paulkakell/sovereign-conquest/pull/12, https://github.com/paulkakell/sovereign-conquest/pull/11.

These are maintenance candidates, not findings that every current dependency is
vulnerable or approval to merge them blindly. Review each diff and test result;
refresh go.mod, go.sum and vendor together for Go dependency changes. Upgrade
builder, CI toolchain and scanner expectations consistently. Re-run module
verification, unit/integration/race/vet/scanners, clean image builds and the
expanded database smoke suite. Treat scan results as time-bound evidence.

For every implemented milestone, update VERSION, config.Version, README release
references, UI fallbacks, smoke-test expected version, image metadata and release
documentation wherever the old literal remains. Use a repository-wide version
reference search, then run version/build contracts. Preserve historical versioned
documents; do not rewrite old release history into the new version.

Record actual implementation commit hashes and PR IDs beside SC-I01-SC-I15 as
they are delivered. The only application commit evaluated in this revision is
03625c68a141f649aa87480b13d61527febceff1. No new application tag or hash is claimed.

## 20. NEXT IMPLEMENTATION CHECKPOINT

Begin with SC-I01 and SC-I02: establish the PostgreSQL fixture and regressions,
then repair BUY demand, repeatable XP, chat fan-out and checked transactional
error handling. Continue SC-I03 account/spawn/reset safeguards and SC-I04 release
promotion before shipping 01.06.04.

The first reviewable implementation should demonstrate a trade route that still
works after repeated replenishment cycles, no progress earned by reading HELP,
correct corporation message delivery, and rollback of failed economic actions.
This gives the later freight and relay systems a dependable base.

Finish M5 before judging the complete discover/trade/cooperate/contest/recover
experience. Finish M6 before opening a scored production season. Use the pilot
results to decide whether M7 adds enough value to justify its scope.

## APPENDIX A. SUPPLIED UNVERSIONED ROADMAP

Historical source follows. Its phase ordering and unresolved alternatives are
superseded by the numbered roadmap above. Preserve this appendix for traceability
and document rollback; do not implement it as a competing current specification.

```text
BEGIN SUPPLIED BASELINE

Product decisions that will matter early

1. Game loop and pacing
   Keep turns. They are the secret sauce. Instead of “per day only,” consider “turns regenerate over time” (for example 1 turn every 2 minutes, with a cap), plus optional server configurations. This preserves planning, supports different play styles, and reduces timezone disadvantage.

2. Universe model
   Use sectors as nodes in a graph. Each sector has edges (warps), optional port, optional planets, hazards, ownership overlays, and player artifacts (fighters, mines). You can keep it procedural at start, but store the generated graph so it becomes shared history.

3. Economy rules
   Ports should buy and sell at varying ratios and replenish on a schedule. Keep three core commodities (ore, organics, equipment) for nostalgia, then add a fourth later only if you need deeper markets. Decide whether prices are purely local (simple) or influenced by global supply (more complex). Local is usually enough to start.

4. Conflict rules
   Trade Wars combat was largely resource and intel driven. Preserve that. Provide tools for scouting, traps, and defensive staging. Avoid real-time twitch. You can modernize with clearer previews and logs, not faster combat.

5. Social layer
   Corporations and diplomacy are where the long-term retention comes from. Build corp chat, shared intel, roles, shared assets, and corp banking early.

Web architecture that fits this game

Client
Single-page web app. Two UI modes tends to work well:

1. Terminal-like command pane for veteran feel.
2. Structured panels for maps, cargo, ports, corp roster, sector intel.

Server
Turn-based games benefit from an authoritative backend that enforces rules and prevents client tampering. A typical stack:

* API: REST or GraphQL for state reads
* Commands: POST /command endpoint or websocket for low latency feel
* Jobs: background workers for port regen, planet production, turn regen, scheduled events
* DB: Postgres

Data modeling (high level)

Core tables

* users
* players (ship stats, credits, turns, sector_id)
* sectors (id, attributes, discovered flags if fog-of-war is per player)
* warps (from_sector, to_sector, one_way bool)
* ports (sector_id, type, inventory, price multipliers, regen schedule)
* planets (sector_id, owner, production, defenses, storage)
* deployables (fighters, mines, limpets, etc)
* corporations, corp_members, corp_assets
* logs (combat, trades, movement, intel)

Two key design calls

* Fog of war: per-player discovery is more interesting. Store discovered sectors and scanned intel by player.
* Determinism: sector graph generation should be reproducible from a seed, but once created, treat it as canonical.

Security and abuse controls

* Every action is validated server-side, consumes turns atomically, and writes an audit log.
* Rate limit commands.
* Use signed sessions (httpOnly cookies) and CSRF protection if cookie auth.
* Anti-bot: simple heuristics and cool-downs first; advanced later.
* Cheating prevention: never trust client-provided outcomes.

MVP scope that still feels like Trade Wars

Phase 1 (playable loop)

* Login, player creation, basic ship
* Sector movement, sector scan
* Ports with buy/sell
* Turns regen and cap
* Simple combat vs sector fighters and 1v1 ship combat
* Logs and event feed
* Basic map view and discovered sectors list

Phase 2 (retention)

* Planets, production, citadel defenses
* Corporations, corp chat, shared assets
* Mines and more trap mechanics
* Rankings, seasons, resets, “soft wipe” options

Phase 3 (modern leverage)

* In-game market analytics, route suggestion (careful: do not kill discovery)
* Events: invasions, anomalies, limited-time sectors
* Mobile-friendly UI

Implementation approach that prevents rewrites later

Command engine
Model the game as commands that transform state. Examples: Move, Scan, TradeBuy, TradeSell, Attack, DeployFighters. Each command:

* loads required state with locks
* validates preconditions
* applies changes in a transaction
* emits events into logs/stream

That single pattern scales cleanly and makes testing easy.

Realtime feel without realtime complexity
Even if the game is turn-based, use websockets or server-sent events to push:

* combat outcomes
* market regen notifications
* corp chat
* sector activity alerts

END SUPPLIED BASELINE
```
