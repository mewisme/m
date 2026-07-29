# override-chain

Fixture for `m explain` showing override-driven version selection and import paths.

- Root depends on `pkg-a@1.0.0`, which depends on `pkg-b@^1.0.0`.
- A global override pins `pkg-b` to `1.0.0` instead of the default `1.2.0`.
