## MODIFIED Requirements

### Requirement: Stream handle lifecycle is well-defined
The generated adaptor code SHALL ensure stream handles are safe and deterministic:
- `Start` SHALL return a non-zero `uint64` handle when successful.
- `Send` and `Finish` SHALL return a non-nil error when the provided handle is unknown, already finished, or otherwise invalid.
- After `Finish` returns (success or error), the handle SHALL become invalid and MUST NOT be reused.
- Concurrent `Send` and `Finish` calls SHALL NOT panic; `Send` after stream end SHALL return a deterministic error.
- Stream end SHALL be signaled via explicit termination (e.g., done/ctx) so the receive path returns `io.EOF` without relying on send channel close.

#### Scenario: Concurrent Send and Finish do not panic
- **GIVEN** a valid stream handle from `Start`
- **WHEN** `Send` and `Finish` are invoked concurrently
- **THEN** no panic occurs
- **AND** `Send` eventually returns a non-nil error once the stream is finished
- **AND** the receive path terminates with `io.EOF`
