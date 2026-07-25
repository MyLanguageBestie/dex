# Why this fork exists

Ora.Cat's iOS app offers Google and Microsoft sign-in, and [App Store Review Guideline
4.8](https://developer.apple.com/app-store/review/guidelines/#sign-in-with-apple) then requires Sign in
with Apple beside them. Upstream Dex cannot do Sign in with Apple, and not for want of configuration:

- Apple **requires `response_mode=form_post`** whenever the authorization request asks for the `name`
  or `email` scope, and then **POSTs** the result to the callback.
- Dex's `/callback` accepts POST only for SAML — it reads `RelayState`, not `state` — and the OAuth2
  path then rejects anything that is not a GET with *"SAML request mapped to OAuth2 connector"*.
- Dex never sends `response_mode` at all ([#2580](https://github.com/dexidp/dex/issues/2580) is open).
- Its OIDC connector also demands a `name` claim in the ID token, which Apple never puts there.

So this is not a configuration gap that a future release closes by accident. The upstream PR that adds
it, [#1905](https://github.com/dexidp/dex/pull/1905), has been open since December 2020 and was parked
pending external-connector support ([#1907](https://github.com/dexidp/dex/issues/1907)) rather than
rejected on merit.

## What is different from upstream

Branch `apple-connector-master`, based on upstream `master`. Deliberately **one commit**, so moving to
a newer Dex is a rebase and the diff stays reviewable.

It tracks master rather than a release tag on purpose: production already runs a master snapshot
(`ghcr.io/dexidp/dex:latest-alpine`), and its Postgres has migrations applied that the last release
tag does not know about. Building on a release would be a *downgrade* against a forward-migrated
schema — on the one service where being wrong locks everybody out.

| File | Change |
|---|---|
| `connector/apple/apple.go` | the connector, ported from PR #1905 |
| `connector/apple/secret/` | mints Apple's client secret from a `.p8` key |
| `connector/connector.go` | adds the `CallbackConnectorFormPOST` interface |
| `server/authflow/login.go` | starts a form_post login |
| `server/authflow/callback.go` | accepts `state` on a POSTed callback, and dispatches it |
| `server/server.go` | registers `"apple"` in `ConnectorsConfig` |

Nothing existing changes behaviour: the new code sits in `case` arms that only a connector implementing
`CallbackConnectorFormPOST` can reach, and the one edit to shared code — reading `state` when
`RelayState` is absent on a POST — previously fell through to an error.

**The client secret takes care of itself.** Apple caps it at 6 months, and the connector re-mints it
from the private key whenever it is close to expiry, so there is no rotation to schedule and no
outage to be surprised by. Give it the `.p8` and the three ids; never a secret.

```yaml
connectors:
  - type: apple
    id: apple
    name: Apple
    config:
      clientID: cat.ora.signin        # the Services ID, not the app's bundle id
      teamID: XXXXXXXXXX
      keyID: YYYYYYYYYY
      privateKeyFile: /etc/dex/apple/AuthKey.p8
      redirectURI: https://auth.ora.cat/callback
```

## Updating from upstream

```bash
git fetch upstream
git rebase upstream/master apple-connector-master
go build ./... && go test ./server/... ./connector/...
git tag oracat-$(git describe --tags --abbrev=0 upstream/master 2>/dev/null || date +%Y%m%d)
git push origin apple-connector-master --tags
```

Retire the fork the day upstream ships an Apple connector, or external connectors land and this can
become a plugin.

## Test status

`go build ./... && go test ./server/... ./connector/...` is green on this branch, vet included. (On the
older `v2.45.1` base two upstream tests failed — an expired SAML fixture and a Go 1.26 vet complaint in
`oauth2.go` — both fixed upstream since.)
