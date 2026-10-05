"""Seeded small NGNN1 fixture plus 64 NGN-produced integer FEN evaluations."""

import argparse
import hashlib
import json
from pathlib import Path

import numpy as np

from trainer.export import export_state
from trainer.refeval import Network
from trainer.tests.synthetic import random_fens


def generate(path: Path, evals: Path, hidden: int = 32, seed: int = 20261005) -> dict:
    rng = np.random.default_rng(seed)
    # The random fixture exercises signs, saturation, and both STM orderings.
    state = {"w1": rng.normal(0, 0.07, (768, hidden)),
             "b1": rng.uniform(-0.1, 0.6, hidden),
             "o": rng.uniform(-0.3, 0.3, 2 * hidden), "ob": np.asarray(-0.031)}
    export_state(state, path)
    network = Network(path)
    fens = random_fens(30, seed + 1)
    fens += ["8/8/8/8/8/8/4Q3/4K2k w - - 0 1", "q3k3/8/8/8/8/8/8/4K3 b - - 0 1"]
    # Side-swapped cases explicitly exercise the second half of output weights.
    fens += [fen.replace(" w ", " b ") if " w " in fen else fen.replace(" b ", " w ") for fen in fens]
    payload = {"format": "NGNN1", "hidden": hidden, "seed": seed,
               "network_sha256": hashlib.sha256(path.read_bytes()).hexdigest(),
               "cases": [{"fen": fen, "eval_cp": network.evaluate_fen(fen)} for fen in fens]}
    evals.parent.mkdir(parents=True, exist_ok=True)
    evals.write_text(json.dumps(payload, indent=2) + "\n")
    return payload


def main() -> None:
    cli = argparse.ArgumentParser(description=__doc__)
    cli.add_argument("--out", type=Path, default=Path("testdata/ngnn1/random_h32.nnue"))
    cli.add_argument("--evals", type=Path, default=Path("testdata/ngnn1/random_h32_evals.json"))
    cli.add_argument("--hidden", type=int, default=32)
    cli.add_argument("--seed", type=int, default=20261005)
    args = cli.parse_args()
    payload = generate(args.out, args.evals, args.hidden, args.seed)
    print(json.dumps({"network": str(args.out), "sha256": payload["network_sha256"], "cases": len(payload["cases"])}))


if __name__ == "__main__":
    main()
