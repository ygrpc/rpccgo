# Introduce a generator-side active record source plan

rpccgo will factor registration planning into a generator-side active record source plan that separates `Origin`, `Contract`, `Transport`, and `Mode`. This keeps the generated runtime output-preserving while making registration sources explicit for Go native, cgo native, cgo message, connect local/remote, and grpc local/remote inputs. The new plan deepens the registration seam without reintroducing runtime dispatch or adapter interfaces.
