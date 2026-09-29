# Pact

Pact is an open-source consumer-driven contract testing framework that records the HTTP (or message) interactions a client expects from a service and replays them against the real provider, so that microservices can verify they are compatible with each other without standing up a full integration environment.

## Primary use cases

- **Replacing brittle end-to-end suites**: instead of spinning up 15 services to check that service A can still call service B, each side is tested in isolation against a shared, versioned contract.
- **Safe independent deploys**: the Pact Broker's `can-i-deploy` check answers "is version X of this service compatible with everything currently in production?" and gates the release pipeline on the answer.
- **API evolution and deprecation**: providers can see exactly which consumers use which fields and endpoints, so they can remove or change something with evidence rather than guesswork.
- **Async messaging contracts**: Pact supports message pacts (Kafka, SQS, RabbitMQ payloads), so event schemas get the same consumer-driven treatment as REST calls.
- **Cross-team boundaries**: where teams own different services and a shared staging environment is constantly broken or contended, contracts give a fast, deterministic signal on each PR.

A team typically adopts Pact when microservice count grows to the point where integration environments are slow and flaky, or after an incident where a "backwards-compatible" API change broke a consumer nobody knew about. It works best with a small number of well-owned consumer/provider relationships, not as a blanket replacement for all integration testing.

## Basic usage

**1. Consumer test (pact-js v4 / PactV4)** generates the contract by running your client code against a mock server:

```ts
import { PactV4, MatchersV3 } from '@pact-foundation/pact';
const { like, integer } = MatchersV3;

const pact = new PactV4({ consumer: 'web-app', provider: 'user-service' });

it('fetches a user', async () => {
  await pact
    .addInteraction()
    .given('user 42 exists')
    .uponReceiving('a request for user 42')
    .withRequest('GET', '/users/42')
    .willRespondWith(200, (b) =>
      b.jsonBody({ id: integer(42), name: like('Ada') }))
    .executeTest(async (mockServer) => {
      const user = await new UserClient(mockServer.url).get(42);
      expect(user.id).toBe(42);
    });
});
// writes ./pacts/web-app-user-service.json
```

**2. Publish the contract to a Pact Broker** (or PactFlow) tagged with the git SHA and branch:

```bash
pact-broker publish ./pacts \
  --consumer-app-version "$(git rev-parse --short HEAD)" \
  --branch "$(git branch --show-current)" \
  --broker-base-url https://broker.example.com --broker-token "$PACT_TOKEN"
```

**3. Verify on the provider side and gate deploys.** The provider replays every consumer's pact against a running instance; `given(...)` states map to setup handlers you write:

```bash
pact-provider-verifier --provider user-service \
  --provider-base-url http://localhost:8080 \
  --pact-broker-base-url https://broker.example.com \
  --provider-app-version "$(git rev-parse --short HEAD)" \
  --publish-verification-results

pact-broker can-i-deploy --pacticipant web-app \
  --version "$(git rev-parse --short HEAD)" --to-environment production
```

## Common pitfalls

- **Over-specified contracts**: asserting exact values instead of matchers (`like`, `regex`, `eachLike`) turns the contract into a snapshot test and causes constant false failures. Specify only what the consumer actually reads.
- **Testing provider business logic**: Pact checks the shape and status of the interface, not correctness of behavior. Keep provider states minimal and use your normal tests for logic.
- **Provider state sprawl**: every `given(...)` needs a setup handler on the provider; unmanaged, these become a large, undocumented fixture layer.
- **Skipping the broker workflow**: passing pact files around manually loses versioning, `can-i-deploy`, and webhooks that trigger provider verification when a contract changes — the features that make Pact valuable at scale. Also remember to record deployments (`record-deployment`) or `can-i-deploy` will give wrong answers.
- **Cultural adoption**: consumer-driven means consumers write the contract, but providers must treat a failing verification as a real signal. Without team buy-in, contracts get ignored or `pending` pacts stay pending forever.
- **Not a schema replacement**: OpenAPI (or bi-directional contract testing in PactFlow) may fit better when consumers are many or external and you can't get them to write pacts.
