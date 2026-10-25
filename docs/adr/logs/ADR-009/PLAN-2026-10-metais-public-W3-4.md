# ADR-009 — PLAN-2026-10-metais-public W3-4: Amendment A12, the UX runner's credential route

**2026-10-25, PLAN-2026-10-metais-public W3-4.** This item builds A12 as
Ed Funnekotter accepted it on 2026-10-25 (the plan's R-W3-3). The
credential is a subscription OAuth token, as `claude setup-token` prints
it, not the API key the proposal named. I1 and I2 (W3-OOB5) run on a host
that has no loopback proxy, and they use this route. The file name carries
the plan's name because earlier trains' items hold the address `W3-4`.

## What landed

- **`-credential-file`** (`internal/tools/ux/route.go`) reads the token
  once, from a pipe, through `os.Stat` and `os.DirFS`, both A8's, at most
  1 KiB and one byte more. It refuses:
  - a path that is not a pipe;
  - a credential that is empty, too long, not one line of printable
    ASCII, or not an `sk-ant-oat` token;
  - the flag beside an `ANTHROPIC_BASE_URL` of the runner's environment,
    or with `-scorecard` or `-scrub`.

  No refusal prints what it read.
- **The route** is an `httputil.ReverseProxy` on `127.0.0.1:0`, alive
  while the set runs. It forwards to `https://api.anthropic.com` alone. It
  drops the session's `x-api-key` and sets `Authorization: Bearer
  <token>`. It merges `oauth-2025-04-20` into the session's own
  `anthropic-beta` values, once. Item 4's refusals are a 403 before
  anything is forwarded: a credential that is not the placeholder, a
  foreign `Host`, a method other than `GET` or `POST`, and a path outside
  `/v1/`. An unclean or escaped path counts as outside `/v1/`.
- **The harness** gains `route`, which makes the route's address the
  `ANTHROPIC_BASE_URL` each session is handed. Each session's environment
  is otherwise unchanged, R-UX-7's placeholder included.
- **`credential-not-seen`** is made after every session while the route
  is on. A run with the token in any file stops the set, naming the file
  but not the token.
- **`layers.json`** gains two `only-importers` rules for item 5. `net` and
  `net/http/...` are imported only by the runner, and by the tests of
  `internal/hostfs` and `internal/guarantees` that already used them.
  `TestLayersControl`'s fixture breaks both rules beneath the runner and
  keeps the runner's exception.

## How it is proved

The headline property was pinned false first. `TestRouteNeverHandsTheCredential`
and `TestRouteRewritesTheCredential` ran red against a route that forwarded
the session's headers unchanged: the fake upstream received the
placeholder. They went green only once the rewrite landed.

- **`TestRouteNeverHandsTheCredential`** runs a faked set of C4 and I1
  through the route to a fake upstream on loopback. Every session reached
  the upstream with the token. No session's environment or arguments held
  it, and no file of the set did.
- **The rewrite** is tested both ways: `TestRouteRewritesTheCredential`
  and `TestRouteBetaNotDoubled`.
- **Every refusal** has a negative test:
  - `TestRouteRefuses`, eleven requests, none of which reaches the
    upstream;
  - `TestReadCredentialRefuses`, on real pipes;
  - `TestCLIRefusesCredentialRoute`;
  - `TestCredentialNotSeen`, a token written to the transcript, the
    session's directory or its `HOME`.
- **Without the route** a run's checks are as before (`TestNoRouteNoCheck`).

No test reaches `api.anthropic.com` or starts a session.

R-W3-3's other half is in the same change. The Opus sessions are given
`opus[1m]` by default. Each record carries its session's `context_windows`
beside `model_requested` and `model_reported`, and the scorecard's Models
row shows all three (`TestOpusModelString`).
