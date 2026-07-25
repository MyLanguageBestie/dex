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

Branch `apple-connector`, based on the release tag it is named after. Deliberately **one commit**, so
that moving to a new Dex release is a rebase and the diff stays reviewable:

| File | Change |
|---|---|
| `connector/apple/apple.go` | the connector, ported from PR #1905 |
| `connector/apple/secret/` | mints Apple's client secret from a `.p8` key |
| `connector/connector.go` | adds the `CallbackConnectorFormPOST` interface |
| `server/handlers.go` | starts a form_post login, and accepts `state` on a POSTed callback |
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
git fetch upstream --tags
git rebase --onto v2.46.0 v2.45.1 apple-connector   # old base -> new base
go build ./... && go test -vet=off ./server/... ./connector/...
git tag oracat-v2.46.0 && git push origin apple-connector --tags
```

Retire the fork the day upstream ships an Apple connector, or external connectors land and this can
become a plugin.

## Known upstream failures, not ours

Both also fail on a pristine `v2.45.1`, so do not go hunting:

- `connector/saml`: `TestVerifyUnsignedMessageAndSignedAssertionWithRootXmlNs` — an expired test
  fixture certificate.
- `server`: `go vet` rejects two `non-constant format string` calls in `oauth2.go` under Go 1.26.
  Use `go test -vet=off ./server/...`.
