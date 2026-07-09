# Generated Stubs — Contract-First Workflow

The OpenAPI specs in `contracts/openapi/*.yaml` are the **source of truth**. Server stubs and
the TS client are generated from them; services conform to the contract, not the reverse.

## Lint the specs
```
npx @redocly/cli@latest lint        # uses ../../redocly.yaml ruleset; must exit 0
```

## Go server stubs (`gen/go/`)
Generated with [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen) v2.4.1.
One package per spec (`types,gin,spec`):
```
cd contracts/gen/go
for s in discover prompts dlp agents compliance graph copilot; do
  mkdir -p $s
  go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.4.1 \
    -generate types,gin,spec -package $s -o $s/$s.gen.go ../../openapi/$s.yaml
done
go mod tidy && go build ./...
```
> oapi-codegen prints a 3.1-support warning but emits compilable Go. Each Go service embeds
> the relevant package and implements its `ServerInterface`.

## TypeScript client (`gen/ts/`)
Generated with [openapi-typescript](https://openapi-ts.dev/) v7 + `openapi-fetch`:
```
cd contracts/gen/ts
npm install
npm run generate      # writes generated/<spec>.ts
npm run typecheck     # tsc --noEmit; must exit 0
```
The frontend imports `makeClients()` from `client.ts` for fully-typed API calls.
