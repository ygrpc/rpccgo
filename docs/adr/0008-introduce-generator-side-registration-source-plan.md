# Introduce a generator-side registration source plan

rpccgo factors registration planning into a generator-side registration source plan that separates `Origin`, `Contract`, `Transport`, and `Mode`. This keeps registration sources explicit for Go native, cgo native, cgo message, connect local/remote, and grpc local/remote inputs while deriving renderer behavior from those source dimensions instead of replacing the source contract with a `RecordRenderer` axis. The plan deepens registration planning without reintroducing adapter interfaces or service-specific runtime registry semantics.
