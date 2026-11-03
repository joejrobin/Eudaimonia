# Research: building a decentralized, Discord-style chat platform

**Status:** decision record, written 2026-11-03. Nothing here is implemented yet; the repo
currently holds only the development environment and a skeleton backend/frontend.

**Decisions taken:**

1. Backend language: **Go** (runner-up: Rust). Rationale in section 5.
2. Decentralization model: **federated community nodes with portable keypair identity**
   (the "relay" model used by Nostr/NIP-29, not Matrix-style server-to-server replication).
   Rationale in section 4. This is the recommended starting point and should be validated
   with a prototype before it hardens into a protocol.
3. Voice: **LiveKit** SFU, one per node, with tokens minted by the node. Section 4.3.
4. Frontend: **React 19 + TypeScript + Vite**. Storage: **PostgreSQL 18**.

---

## 1. What "Discord-like" has to mean

The feature set below is what users will measure the product against. Any architecture
that cannot deliver one of these is a non-starter, which is what rules out most pure
peer-to-peer designs.

| Feature | Notes for a decentralized design |
|---|---|
| Communities ("servers"/guilds) with many channels | A community is the natural unit of hosting and moderation. |
| Roles and permissions per community | Must be enforced somewhere trusted by members, i.e. by the host of the community. |
| Persistent, searchable history, available while you were offline | Needs an always-on party holding the history. |
| Presence and typing indicators | Cheap in a hub model, expensive in gossip/P2P. |
| Always-on voice channels you can drop into | Needs an SFU with a public address, not browser-to-browser mesh. |
| Direct messages and small group DMs | The place where end-to-end encryption matters most. |
| Invites, bans, slow mode, audit log | Moderation tooling is what keeps communities alive. |
| Media uploads, link previews, emoji, reactions, threads | Mostly application-level; needs blob storage per node. |

## 2. The four ways to decentralize chat

| Model | Examples | How it works | Fit for Discord-like |
|---|---|---|---|
| **Federated, server-to-server replication** | Matrix, XMPP, ActivityPub | Users have a home server; servers replicate room state to each other. | Best-proven for large communities, but the replication/state-resolution machinery (Matrix's DAG + state resolution) is the hardest part of the whole system to build and debug. |
| **Relay / multi-home** | Nostr (NIP-29 groups), roughly AT Protocol's PDS model | Identity is a keypair, not an account on a server. Clients talk directly to whichever nodes host the communities they care about. Nodes do not replicate to each other. | Maps almost 1:1 onto Discord: node = community host, group = channel. Dramatically simpler than S2S federation. Weakness: a community lives on one node unless you add mirroring. |
| **Peer-to-peer** | Keet (Hypercore/Pear), Berty and Quiet (libp2p/IPFS), iroh-based apps | No servers; devices sync logs directly, with relays only for NAT traversal. | Excellent for private groups and DMs; poor for large always-on communities, history for new joiners, moderation, and voice rooms. |
| **Blockchain-adjacent** | Status Communities (Waku) | Gossip network plus on-chain identity/tokens. | Adds token and incentive complexity we do not need. |

## 3. Prior art, as of November 2026

**Matrix.** Still the most complete decentralized chat protocol and the one every "leave
Discord" guide recommends for decentralization and E2EE. Server landscape: Synapse (Python,
Element) is the default and runs roughly three quarters of federating servers in the
September 2026 federation stats; Dendrite (Go) is in maintenance mode with security fixes
only; `conduwuit` (Rust) was archived in May 2026 and split into **tuwunel** (full-time
maintained) and **continuwuity** (community fork). Matrix 2.0's Simplified Sliding Sync is
native in Synapse since 1.114, and MatrixRTC (Element Call) is built on **LiveKit** with a
small JWT service that mints LiveKit tokens after Matrix auth. The recurring complaints in
the guides are onboarding friction and the absence of Discord-style always-on voice
channels out of the box. Building *on* Matrix is viable (our backend would shrink to a
homeserver plus glue), but it means inheriting the full spec and mapping Discord concepts
onto Spaces, rooms and power levels.

**Stoat (formerly Revolt).** The most Discord-like open-source clone, Rust backend, but
centralized (self-hostable, no federation), pre-1.0, no E2EE, voice self-hosting still
undocumented per the 2026 roundups.

**Spacebar.** TypeScript reimplementation of the Discord client/server API. Self-hostable,
Discord clients can connect, but no voice and no federation between instances. Useful as a
reference for *what the Discord API surface looks like*, not as a base.

**Nostr NIP-29 ("relay-based groups").** Draft, optional NIP. A group is just an id that a
relay enforces rules for; the relay signs the group's metadata (`kind:39000`), admin list
(`39001`), member list (`39002`) and role list (`39003`). Membership changes are moderation
events in the `9000`-`9020` range, and any content kind can be posted into a group with an
`h` tag. Notably the metadata already carries a `livekit` flag and a `39004` participants
event, so the "voice via LiveKit, token issued by the host" pattern is already in the spec.
Clients that treat this as a Discord: **Flotilla** (one relay = one community, groups =
channels), **Chachi** (many communities per relay), 0xChat. Server side in Go: **khatru**
(relay framework; the GitHub repo was archived January 2026 and development continues at
the module path `fiatjaf.com/nostr/khatru`, last published November 2026) and **relay29**
(NIP-29 on top of khatru, last pushed 2024 and self-described as "probably broken"). This
is the model that most closely matches what we want to build, and the thinnest to implement.

**Peer-to-peer stacks.** **Keet** (Holepunch, Pear runtime, Hypercore) ships 4.16.x as of
June 2026 and claims very large rooms, but is a product, not an open protocol we can build
against. **iroh 1.0** (June 2026) is the nicest P2P transport available today: dial by
public key, QUIC, hole-punching that mostly just works, official bindings for Python,
Node, Swift and Kotlin, but it is Rust-first, has no Go bindings, and intentionally lacks
a Kademlia DHT. **libp2p** remains the flexible, multi-language option (Go is the
reference implementation) at the cost of configuring everything yourself. Neither solves
the always-on community problem; both are candidates for a later "mirror a community
between nodes" or "device-to-device DM" feature.

**Voice/video SFUs.** **LiveKit** (Apache-2.0, Go, built on Pion) is the default choice in
2026 write-ups: single binary, Docker/Kubernetes friendly, simulcast, recording, first-party
server SDKs including Go, and it is what Matrix standardized on. **mediasoup** (C++ core,
Node API) shows roughly double the participant density in OpenVidu's benchmarks, but those
are vendor-run and video-heavy, and mediasoup leaves signaling entirely to you. **ion-sfu**
appears unmaintained. For voice channels LiveKit is the pragmatic pick; density can be
revisited if it ever matters.

## 4. Recommended architecture: community nodes + portable identity

### 4.1 Shape

```
            ┌──────────────┐        ┌──────────────┐
  user key  │  node A      │        │  node B      │   each node hosts
  (client)──┤  community 1 │   …    │  community 3 │   N communities
     │      │  community 2 │        │              │
     │      │  Postgres    │        │  Postgres    │
     │      │  LiveKit SFU │        │  LiveKit SFU │
     │      └──────────────┘        └──────────────┘
     └────── one WebSocket per node the user has communities on ──────┘
```

* **Identity is a keypair held by the client.** Every event (message, join, role change)
  is signed by its author. A user is the same person on every node; nothing is "homed".
* **A node is the host of one or more communities.** It stores history, enforces roles and
  permissions, rate-limits, runs the SFU, and serves media. Anyone can run a node (that is
  the decentralization), and a community can be exported and re-hosted elsewhere because
  all state is a log of signed events.
* **Clients multi-home.** The client holds one connection per node and merges the result
  into the familiar left-hand community list. Discovery is initially invite links
  (`eudaimonia://node.example/<community-id>?invite=…`); a directory/DHT can come later.
* **No server-to-server protocol in v1.** This is the single biggest simplification versus
  Matrix. Mirroring a community across nodes (for resilience or censorship resistance) is
  an explicit later phase and can be layered on top of the signed event log.

### 4.2 Discord concept mapping

| Discord | Here | Enforced by |
|---|---|---|
| Server / guild | Community, hosted on exactly one node (v1) | node |
| Channel, category | Group (optionally with a parent group) | node |
| Role, permission overwrite | Signed role-assignment event; role definitions published by node | node |
| Member list, presence | Node-signed membership list; presence from live connections | node |
| Message, reaction, thread | Signed content event referencing the group | author signs, node stores |
| Voice channel | LiveKit room per group; node mints join token after permission check | node + SFU |
| DM / group DM | Encrypted event between keys, stored on any node both parties use | end-to-end crypto |
| Invite link | Node-issued invite code bound to a community | node |
| Ban / kick / timeout | Moderation event from a role holder, applied by node | node |

### 4.3 Voice

Each node runs a LiveKit server. When a client wants to join a voice group, it asks its node
over the authenticated WebSocket; the node checks the role/permission, then signs a
short-lived LiveKit JWT with the node's LiveKit API key and returns it together with the
SFU URL. This is exactly the shape of Matrix's `lk-jwt-service`, and of the `livekit` flag
in NIP-29. The browser then talks WebRTC to the SFU directly.

### 4.4 Open protocol question: speak NIP-29 on the wire, or not?

Two defensible paths, to be settled by a short spike:

* **NIP-29-compatible.** Adopt Nostr's event envelope (secp256k1 Schnorr signatures, kinds,
  tags) and NIP-29 semantics. Win: existing clients (Flotilla, Chachi, 0xChat), relay
  tooling (khatru, eventstore) and key management (NIP-07 browser extensions, nsec
  bunkers) work on day one. Cost: a draft spec, Nostr's quirks (JSON-array tags, no schema),
  and a dependency on an ecosystem whose culture may not match the product.
* **Bespoke, NIP-29-inspired.** Same architecture, our own typed protocol (likely JSON or
  protobuf over WebSocket), Ed25519 keys, explicit schemas. Win: cleaner, easier to
  evolve, easier permission model. Cost: no interop, every client is ours.

Recommendation: start bespoke but keep the event model structurally compatible (signed
events, group tag, node-signed metadata, moderation-as-events), so a NIP-29 bridge remains
cheap if interop turns out to matter.

### 4.5 Deferred, deliberately

End-to-end encryption for communities (large-group E2EE means MLS; Matrix is still
rolling it out), cross-node mirroring, a global directory, and native mobile clients.
DM encryption should land early because it is the privacy expectation users bring.

## 5. Language decision

The design above is an HTTP/WebSocket service with a Postgres store, signature
verification on every event, fan-out to many long-lived connections, and tight
integration with an SFU. The candidates were Go, Rust, Elixir and TypeScript.

| Criterion | Go | Rust | Elixir | TypeScript |
|---|---|---|---|---|
| Ecosystem fit for *this* design | LiveKit server and SDK are Go; khatru/relay29 are Go; go-libp2p is the reference libp2p; Dendrite proved federated chat in Go | tuwunel/continuwuity, iroh 1.0, matrix-sdk, rust-nostr: strongest if P2P transport becomes central | Phoenix Channels/Presence are the best primitives for fan-out; Pleroma proved federation; little for Nostr/WebRTC | matrix-js-sdk and nostr-tools are the most used client libs; Spacebar; shares types with the React app |
| Long-lived WebSocket fan-out | Goroutine per connection, fine to 100k+ per node | Excellent, tokio | Best in class, BEAM was built for it | Needs clustering to use multiple cores |
| Deploy footprint | Static binary, ~15 MB distroless image, trivial cross-compile | Static binary, small image, slow builds unless cached (cargo-chef) | Release tarballs, larger images, Erlang runtime | Node image plus `node_modules` |
| Iteration speed for a small team | Fast compile, small language, strong std lib (`net/http`, `log/slog`, `crypto/ed25519`) | Slowest to iterate; borrow checker friction in shared-state servers | Fast once learned; smaller talent pool | Fastest to start; weakest for crypto/network-heavy infra |
| Tooling in devenv/Nix | First-class `languages.go`, version pinning | First-class | First-class | First-class |

**Decision: Go.** The deciding factors are that the voice stack (LiveKit, Pion) and the
closest prior art for our exact architecture (khatru, relay29, Dendrite) are all Go, that a
single static binary keeps per-node self-hosting trivial (which is the whole point of
decentralization), and that Go's iteration speed suits a protocol that is still being
designed. Rust is the runner-up and the right call if the roadmap flips toward
device-to-device transport, where iroh has no Go equivalent. Elixir would win a pure
"most concurrent connections per box" contest but loses on every ecosystem row that
matters here. TypeScript stays on the frontend.

Benchmarks circulating online that rank these languages for chat servers contradict each
other and were not used; the choice is about ecosystem and operability, not raw throughput.

## 6. Tooling picked

| Layer | Choice | Version basis (checked 2026-11-03) |
|---|---|---|
| Backend language | Go | 1.27.3 is the newest stable on go.dev/dl |
| HTTP routing | standard library `net/http` with method patterns (Go ≥ 1.22) | no framework until one is earned |
| WebSocket | `github.com/coder/websocket` (successor of nhooyr) | add when realtime work starts |
| Database | PostgreSQL 18; `pgx` driver; `sqlc` for typed queries; `goose` for migrations | |
| Logging | `log/slog` JSON | |
| Lint/format | `gofmt`, `golangci-lint`; pre-commit via devenv git-hooks | |
| Live reload | `air` (runs under `devenv up`) | |
| Voice | LiveKit server (Docker image `livekit/livekit-server`), Go server SDK for tokens | |
| Frontend | React 19, TypeScript, Vite, served by nginx in production | scaffolded with `create-vite` |
| Node runtime | Node 26 (Active LTS since 2026-10-28) | nodejs/Release schedule |
| Dev environment | devenv 2.4 (`languages.go`, `languages.javascript`, `services.postgres`, `processes`) | installed locally |
| Containers | Multi-stage Dockerfiles (distroless static for Go, nginx-alpine for the SPA), `compose.yaml` with Postgres and LiveKit | Docker is not installed on this machine; the files are unverified by a real build |

## 7. Next steps

1. Spike: a node that accepts signed events for one community over WebSocket, stores them
   in Postgres, and replays history to a second client. Decide NIP-29 compatibility
   (section 4.4) on the basis of that spike.
2. Identity: Ed25519 keys generated in the browser, stored in IndexedDB, with an export/
   import flow; evaluate passkeys/WebAuthn PRF for key protection.
3. Voice: wire LiveKit token minting into the node and a join/leave UI in the React app.
4. Moderation primitives (roles, bans, invites) as signed events, with an audit log view.
5. Encrypted DMs.

## Sources

Decentralization landscape and Discord alternatives:
[Zap-Hosting self-hosted alternatives ranking](https://zap-hosting.com/en/blog/2026/02/the-best-self-hosted-discord-alternatives-2026-ranking-pros-cons/),
[Stoat Board open-source alternatives](https://stoatboard.com/guides/best-open-source-discord-alternatives),
[Software for Good: secure communities guide](https://softwareforgood.com/its-time-to-leave-discord-heres-where-to-go/),
[TechFuel hardware-grounded comparison](https://techfuelhq.com/articles/self-hosted-discord-alternatives-2026/).

Matrix:
[Matrix homeservers overview](https://matrixdocs.github.io/docs/servers/overview),
[This Week in Matrix 2026-09-18 (federation stats)](https://matrix.org/blog/2026/09/18/this-week-in-matrix-2026-09-18/),
[This Week in Matrix 2026-07-17](https://www.matrix.org/blog/2026/07/17/this-week-in-matrix-2026-07-17/),
[This Week in Matrix 2026-10-02 (sliding sync MSCs, Element Call)](https://matrix.org/blog/2026/10/02/this-week-in-matrix-2026-10-02/),
[conduwuit archive notice](https://github.com/x86pup/conduwuit),
[tuwunel install/migration notes](https://hysenlabs.com/projects/matrix-construct-tuwunel),
[Matrix 2.0 is here](https://www.matrix.org/blog/2024/10/29/matrix-2.0-is-here),
[continuwuity LiveKit/MatrixRTC guide](https://g.lair.moe/Mirrors/continuwuity/blame/branch/ginger/remove-sliding-sync-proxy/docs/calls/livekit.mdx).

Nostr / NIP-29:
[NIP-29 specification](https://github.com/nostr-protocol/nips/blob/master/29.md),
[khatru (new module path)](https://pkg.go.dev/fiatjaf.com/nostr/khatru),
[khatru (archived GitHub repo)](https://github.com/fiatjaf/khatru),
[relay29](https://github.com/fiatjaf/relay29),
[Flotilla](https://awesome.ecosyste.ms/projects/github.com%2Fcoracle-social%2Fflotilla),
[Zooid community relay](https://gitea.coracle.social/triesap/zooid).

Spacebar: [server repository](https://github.com/SpacebarChat/Server), [docs](https://docs.spacebar.chat/).

Peer-to-peer:
[Comparing iroh and libp2p](https://www.iroh.computer/blog/comparing-iroh-and-libp2p),
[iroh FAQ](https://docs.iroh.computer/about/faq),
[iroh 1.0 release coverage](https://www.techtimes.com/articles/318490/20260616/peer-peer-library-iroh-10-ships-dial-devices-key-not-ip-address.htm),
[Why iroh, not libp2p or WebRTC (practitioner write-up)](https://www.uniclipboard.app/blog/why-iroh-not-libp2p-webrtc),
[Keet on Snapcraft (release 4.16.3)](https://snapcraft.io/keet),
[Holepunch P2P stack overview](https://tianpan.co/forum/t/holepunchs-p2p-stack-building-unstoppable-apps-without-infrastructure/677).

Voice:
[WebRTC media infrastructure 2026 deep dive](https://www.youngju.dev/blog/culture/2026-05-14-webrtc-media-infrastructure-2026-livekit-pion-daily-100ms-mediasoup-whip-whep-deep-dive.en),
[OpenVidu performance benchmarks (mediasoup vs Pion)](https://openvidu.io/3.8/docs/self-hosting/production-ready/performance/),
[LiveKit self-hosting deployment docs](https://docs.livekit.io/transport/self-hosting/deployment/).

Languages and runtimes:
[Go downloads JSON (stable versions)](https://go.dev/dl/?mode=json),
[Go 1.27 release notes](https://go.dev/doc/go1.27),
[Node.js release schedule](https://github.com/nodejs/Release),
[Go vs Rust vs Elixir commentary](https://medium.com/@tl77/go-vs-rust-vs-elixir-for-your-api-d43f6793cceb),
[Why I chose Elixir over Go and Rust](https://dev.to/setas/why-i-chose-elixir-over-go-and-rust-for-my-cloud-platform-2058).

devenv: [languages](https://devenv.sh/languages/), [services](https://devenv.sh/services/),
[postgres](https://devenv.sh/services/postgres/), [containers](https://devenv.sh/containers/),
[postgres example](https://github.com/cachix/devenv/blob/main/examples/postgres/devenv.nix).
