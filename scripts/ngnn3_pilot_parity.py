"""Select 20 owned pilot records for NGNN3 integer export/engine parity."""

import argparse
import hashlib
import json
from pathlib import Path

import numpy as np

from trainer.ngnp import Shards, decode_records, decode_record, to_fen
from trainer.refeval import Network


def main():
    cli = argparse.ArgumentParser(description=__doc__)
    cli.add_argument("--net", type=Path, required=True)
    cli.add_argument("--data", nargs="+", required=True)
    cli.add_argument("--out", type=Path, required=True)
    args = cli.parse_args()
    network = Network(args.net)
    if network.version != 3:
        cli.error("Expected NGNN3")
    shards = Shards(args.data, require_kings=True)
    by_material, by_context = {}, {}
    for records in shards.maps:
        for offset in range(0, len(records), 16384):
            block = records[offset:offset+16384]
            block = block[shards.mask(block)]
            if not len(block):
                continue
            board = decode_records(block)["board"]
            counts = (board >= 0).sum(axis=1)
            buckets = np.clip((counts-2)*network.buckets//32, 0, network.buckets-1)
            white_king = (board == 5).argmax(axis=1)
            black_king = (board == 11).argmax(axis=1) ^ 56
            white_mirror = (white_king % 8 >= 4).astype(np.int64)*7
            black_mirror = (black_king % 8 >= 4).astype(np.int64)*7
            table = np.asarray(network.king_map)
            contexts = np.stack((buckets, table[white_king ^ white_mirror],
                                 table[black_king ^ black_mirror], white_mirror, black_mirror), axis=1)
            for bucket in np.unique(buckets):
                by_material.setdefault(int(bucket), block[np.flatnonzero(buckets == bucket)[0]].copy())
            _, indices = np.unique(contexts, axis=0, return_index=True)
            for index in indices:
                key = tuple(map(int, contexts[index]))
                if len(by_context) < 64:
                    by_context.setdefault(key, block[index].copy())
        if len(by_material) == network.buckets and len(by_context) >= 20:
            break
    selected = list(by_material.values())
    used = {record.tobytes() for record in selected}
    for record in by_context.values():
        if record.tobytes() not in used:
            selected.append(record)
            used.add(record.tobytes())
        if len(selected) >= 20:
            break
    if len(selected) != 20:
        raise RuntimeError("Not enough distinct pilot parity positions")
    cases = []
    for record in selected:
        pos = decode_record(record)
        cases.append({"fen": to_fen(pos), "eval_cp": network.evaluate(pos), "bucket": network.bucket(pos)})
    payload = {"format": "NGNN3", "hidden": network.hidden, "buckets": network.buckets,
               "king_buckets": network.king_buckets,
               "network_sha256": hashlib.sha256(args.net.read_bytes()).hexdigest(), "cases": cases}
    args.out.write_text(json.dumps(payload, indent=2)+"\n")
    print(json.dumps({"cases": len(cases), "material_buckets": sorted(by_material),
                      "network_sha256": payload["network_sha256"]}), flush=True)


if __name__ == "__main__":
    main()
