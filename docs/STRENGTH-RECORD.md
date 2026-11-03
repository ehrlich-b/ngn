# Strength record v1

A strength record is one JSON object per completed game. `internal/rating.ReadStrengthRecord` reads one object, rejects unknown, noncanonical, and duplicate fields and validates version 1. The examples in `internal/rating/testdata` are illustrative, not measurements or claims about actual games.

`engine` and `opponent` each have a nonempty `name` and `version`, positive `threads`, and `source`, `binary`, and `model` provenance. Each provenance object has `status`: `known` with a nonempty `value`, or `unknown` with no value. Source values identify a revision; binary and model values are lowercase SHA-256 hex digests of the actual bytes. Use unknown when evidence is unavailable, including when no model identity is known; never substitute a label or an empty hash. The two name/version pairs must differ so their roles are unambiguous.

`time_control` describes the full control (including units or conventional seconds+increment notation), `opening` identifies the starting position or opening sequence, and `engine_color` is `white` or `black`. `result` is the PGN result from White's perspective (`1-0`, `0-1`, or `1/2-1/2`); `termination` describes why the game ended. These strings should preserve enough detail to interpret the game.

The optional `opponent_historical_rating` has a `system` and positive integer `value`. It is a historical label, not a score measured by this game. The game result is the measured outcome; no Elo estimate is implied. Consumers must not conflate the two. Future incompatible changes require a new schema version.
