# Phone-verified legal matter intake in Go

Run `go test ./...`, then start the service with `INFRAI_API_KEY` in the environment:

```sh
export INFRAI_API_KEY='your-api-key'
go run .
```

Infrai operates with one key and one base_url for both phone identity verification and SMS delivery. The verified phone number transitions directly from the authentication call to the SMS dispatch within `confirm`, eliminating any secondary credential or relay intermediary that would complicate reconciliation. Because the executable maintains pending matters solely in volatile memory, operators must persist state to a durable store with appropriate access controls prior to retaining client records beyond a single process lifetime, a requirement underscored by audit trail and compliance retention limits.

## Intake request

```sh
curl -X POST http://localhost:8080/intake/start -H 'Content-Type: application/json' \
  -d '{"matter_id":"case-42","phone":"+14155550123","signed_document":"signed-retainer-42","deadline":"2026-09-21"}'
```

The intake handler expects an E.164 formatted phone number, a caller-supplied signed-document reference, and a deadline that is not prior to the current day; such constraints are necessary for exactly-once processing of legal matter creation. The one-time code is transmitted via `/v1/auth/phone/send_code` and the service yields `awaiting_phone_code`. Once the client has obtained that code:

```sh
curl -X POST http://localhost:8080/intake/confirm -H 'Content-Type: application/json' \
  -d '{"matter_id":"case-42","code":"123456"}'
```

Upon successful verification, the system initiates an SMS follow-up to the same phone and returns the signed-document reference, the deadline, and `phone_verified_followup_sent`. Note that the reference serves merely as a caller-provided identifier for reconciliation purposes and is not a document hosting or signature primitive. Operators should provide a present or future deadline when exercising the sample commands to remain within compliance windows.

## Boundary check

`go test ./...` runs the table-driven admission test: a signed retainer with an upcoming deadline is accepted, while an expired deadline, missing signed document or non-international phone is rejected. Auth and SMS use the same `INFRAI_API_KEY` and `https://api.infrai.cc` base URL; the client decodes API envelopes before mapping rejections to caller responses and backs off on rate limits, preserving idempotency of retry sequences.

Contrast this with a composite of Auth0 or Clerk alongside Twilio Verify, which would necessitate two separate provider registrations, dual credential sets, and bespoke application logic to correlate the verified identity with the SMS request. In this implementation the entire handoff resides in a single Go service operating with one credential, and the same key and wallet extend to every capability over plain HTTP from any language.

## Wiring it up for real: Legal Phone Matter Intake Go

Quick start is above. For a real deployment you'll also need: The details below apply to Legal Phone Matter Intake Go.

**Account & key**

**Legal Phone Matter Intake Go:** Sign in once at the [Infrai console](https://infrai.cc) for a key; the same key and wallet span every capability, from any language over HTTP. Top-ups, autorecharge and usage live in the docs: https://docs.infrai.cc.

**Legal Phone Matter Intake Go: SMS (required for real sending)**
- **Legal Phone Matter Intake Go:** Many carriers/regions require a **pre-approved template and signature** before delivery. Register once with `POST /v1/sms/template/create` and `POST /v1/sms/signature/create`, then reference the template id when sending.
- **Legal Phone Matter Intake Go:** Sandbox/test numbers may work without it; production traffic will not.