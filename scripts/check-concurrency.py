#!/usr/bin/env python3
"""One real max-two check: cancel one isolated trial while its peer finishes."""
import json
import os
from pathlib import Path
import signal
import subprocess
import time
import uuid

ROOT = Path(__file__).resolve().parent.parent
os.chdir(ROOT)
owner = uuid.uuid4().hex
sentinel = None
active = None
summary = {}


def command(*args):
    return subprocess.check_output(args, text=True, timeout=20).strip()


try:
    sentinel = command("docker", "create", "--name", "itl-pair-sentinel-" + owner[:12],
                       "--label", "incident-trial-lab.pair-test-owner=" + owner,
                       "kindest/node@sha256:099e049362a1526b2db71494e1947aae99bd16290d7c895f2b7ea312e3cbfaed",
                       "/bin/sleep", "300")
    active = subprocess.Popen([".build/trial-batch", "-workers", "2", "-variants", "fixed,fixed"],
                              stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
    first = active.stdout.readline()
    assert first.startswith("Batch evidence: "), first
    bundle = Path(first.strip().split(": ", 1)[1])
    records = {}
    deadline = time.monotonic() + 150
    while time.monotonic() < deadline:
        assert active.poll() is None, "Batch ended before paired checkpoint"
        for job in ("job-1", "job-2"):
            log = bundle / (job + ".log")
            if not log.exists():
                continue
            lines = log.read_text().splitlines()
            if not lines or not lines[0].startswith("Lifecycle run "):
                continue
            run_id = lines[0].split()[2].rstrip(";")
            root = Path("evidence") / ("lifecycle-" + run_id)
            record = json.loads((root / "record.json").read_text())
            forwards = list(root.rglob("port-forward.log"))
            if any("Forwarding from 127.0.0.1:" + str(record["port"]) in p.read_text() for p in forwards):
                records[job] = (run_id, root, record)
        if len(records) == 2:
            break
        time.sleep(0.1)
    else:
        raise RuntimeError("Both trials did not overlap at port-forward checkpoint")
    assert records["job-1"][0] != records["job-2"][0]
    assert records["job-1"][2]["port"] != records["job-2"][2]["port"]
    nodes = {}
    for job, (_, root, _) in records.items():
        snapshots = list(root.rglob("ownership.json"))
        assert len(snapshots) == 1
        nodes[job] = json.loads(snapshots[0].read_text())
        assert command("docker", "inspect", "--format", "{{.State.Running}}", nodes[job]["id"]) == "true"
    # Cancellation request belongs only to job-1 in this unique batch directory.
    (bundle / "cancel" / "job-1").write_text("cancel\n")
    output, _ = active.communicate(timeout=120)
    exit_code = active.returncode
    active = None
    (bundle / "paired-check.log").write_text(first + output)
    result = json.loads((bundle / "batch.json").read_text())
    assert result["workers"] == 2 and exit_code == 1
    one, two = result["results"]
    assert one["status"] == "canceled" and one["business_verdict"] == "inconclusive", one
    assert two["status"] == "completed" and two["business_verdict"] == "passed", two
    assert one["cleanup_verified"] and two["cleanup_verified"]
    for job, (run_id, root, _) in records.items():
        assert not command("docker", "ps", "-aq", "--filter", "label=io.x-k8s.kind.cluster=" + nodes[job]["cluster"])
        assert not list(root.rglob("kubeconfig"))
        recovered = subprocess.run([".build/trial-recover", "-run-id", run_id], text=True, capture_output=True, timeout=65)
        assert recovered.returncode == 0, recovered.stdout + recovered.stderr
    assert command("docker", "inspect", "--format", "{{.State.Status}}", sentinel) == "created"
    summary = {"batch": result, "simultaneous_live_nodes_verified": True,
               "distinct_ports": [records[j][2]["port"] for j in ("job-1", "job-2")],
               "distinct_node_ids": [nodes[j]["id"] for j in ("job-1", "job-2")],
               "reconciliation_compatible": True, "sentinel_preserved": True,
               "evidence_path": str(bundle)}
    (bundle / "paired-summary.json").write_text(json.dumps(summary, indent=2) + "\n")
    print(json.dumps(summary, indent=2), flush=True)
finally:
    if active is not None and active.poll() is None:
        active.send_signal(signal.SIGTERM)
        active.wait(timeout=120)
    if sentinel is not None:
        labels = json.loads(command("docker", "inspect", "--format", "{{json .Config.Labels}}", sentinel))
        assert labels.get("incident-trial-lab.pair-test-owner") == owner
        command("docker", "rm", sentinel)
