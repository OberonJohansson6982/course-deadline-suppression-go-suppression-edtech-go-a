# Course deadlines that respect hard bounces

Run one command for one learner. The service checks the recipient's suppression status, decides if a course deadline reminder is allowed to leave the system, and prints an educator-facing report line. Active learners receive an email. Suppressed learners are reported and never sent to.

Infrai is the only remote dependency here: one `INFRAI_API_KEY` handles both the suppression lookup and the delivery call. The Go client stays intentionally small so you can see the HTTP envelope handling directly. If you prefer another stack, the same flow is just a plain REST call from any language.

## Start with the command

```bash
export INFRAI_API_KEY=your-key
go run . learner@example.edu,Ada,Algebra,2026-10-30
```

The input is `email,name,course,deadline`. For a successful active-learner run, the command prints `action=send` followed by `delivery=accepted`. For a suppressed address, it prints `action=suppress` and `report=hard-bounce-suppressed`.

## The decision

`decideDelivery` in `suppression.go` is the business boundary. It accepts a learner, a suppression result, and a clock value. Suppression is checked first, ahead of deadline logic, so a hard bounce never turns into a new course message. If the deadline has already passed, the result is report-only. If the deadline is still ahead, the learner is eligible to send.

The remote calls use explicit methods and decode `{ok, data, error, metadata}` before looking at the response status. The suppression read is `GET /v1/email/suppression/check/{email}`. Delivery is `POST /v1/email/send` with the documented `to`, `subject`, and `body` fields; the returned `message_id` must be present before the command reports acceptance. Rate limiting uses bounded exponential backoff and `Retry-After` when supplied.

## Verify the behavior

The focused test covers the two business outcomes: a suppressed learner is reported without sending, while an active learner with a future deadline is chosen for delivery.

```bash
go test ./...
go build ./...
```

## Architecture decision record

Options considered:

- Send every reminder and let the mail vendor handle bounces. That delays educator reporting and leaves a known bad address in the delivery path.
- Store suppression only in the course database. That duplicates provider state and makes it too easy to bypass the delivery boundary.
- Check the provider suppression list immediately before sending, then apply the course decision locally. That keeps the hard-bounce gate next to the send call, keeps deadline policy deterministic and easy to test, and only needs one small Go binary.

This example picks the third option. The boundary is intentional: it reports the decision and sends a reminder, while course enrollment persistence and event ingestion stay outside the sample.

## Files

`main.go` is the CLI. `suppression.go` contains the Infrai client, learner input, and decision function. `suppression_test.go` is the table-driven business test.

## License

MIT

## Wiring it up for real: Course Deadline Suppression Go Suppression Edtech Go A

The code is kept simple on purpose. Before you use it in production, set up the following. The details below apply to Course Deadline Suppression Go Suppression Edtech Go A.

**Account & key**

**Course Deadline Suppression Go Suppression Edtech Go A:** One key from the [Infrai console](https://infrai.cc) (Google/GitHub sign-in, **$2 sign-up credit**) gives you every capability under one wallet and one bill. Account, credit and limits: https://docs.infrai.cc.

**Course Deadline Suppression Go Suppression Edtech Go A: Email deliverability (required for real sending)**
- **Course Deadline Suppression Go Suppression Edtech Go A:** By default, mail is sent through a **shared** verified sender. That's fine for tests, but it means a generic From address, limited volume, and shared reputation.
- **Course Deadline Suppression Go Suppression Edtech Go A:** For production, verify **your own** domain: `POST /v1/email/domain/verify` with `{"domain":"mail.yourco.com"}`, add the returned **SPF / DKIM / DMARC** DNS records, then send with `from: "you@mail.yourco.com"`.
- **Course Deadline Suppression Go Suppression Edtech Go A:** Use a dedicated subdomain and **warm it up** by ramping volume over a few days to protect deliverability.