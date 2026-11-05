"""Check the committed report; --mutate-division must fail this same check."""

import argparse
import json
from pathlib import Path

from trainer.quantized_validation import evaluate_stream, prediction_stream, record_stream
import trainer.refeval as reference


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--mutate-division", action="store_true")
    args = parser.parse_args()
    fixture = Path(__file__).parent
    if args.mutate_division:
        reference.trunc_div = lambda n, d: n // d
    records = record_stream(fixture / "records.ngnp")
    predictions = prediction_stream(fixture / "float-cp.jsonl")
    try:
        actual = evaluate_stream(reference.Network(fixture / "h32.nnue"), records,
                                 max_records=32, float_predictions=predictions)
    finally:
        records.close()
        predictions.close()
    expected = json.loads((fixture / "report.json").read_text())
    if actual != expected:
        raise AssertionError("Integer diagnostic differs from committed hand-checked fixture report")
    print("32-record H32 report matches; QA=255 QB=64 score_scale=400")


if __name__ == "__main__":
    main()
