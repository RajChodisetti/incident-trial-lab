#!/usr/bin/env python3
"""Two real SIGKILL/restart checks, without restarting shared Docker."""
import datetime
import json
import os
from pathlib import Path
import signal
import subprocess
import time
import uuid

ROOT = Path(__file__).resolve().parent.parent
os.chdir(ROOT)
bundle = Path("evidence") / ("recovery-validation-" + datetime.datetime.now(datetime.timezone.utc).strftime("%Y%m%dT%H%M%SZ"))
bundle.mkdir(parents=True, exist_ok=False)
owner = uuid.uuid4().hex
sentinel = None
active = None
results = {}


def command(*args):
    return subprocess.check_output(args, text=True, timeout=20).strip()


def check_case(name, kill_children):
    global active
    active = subprocess.Popen([".build/trial-run", "-mode", "candidate", "-variant", "fixed", "-timeout", "120s"],
                              stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
    first = active.stdout.readline()
    if not first.startswith("Lifecycle run "):
        raise RuntimeError("Supervisor did not start: " + first)
    run_id = first.split()[2].rstrip(";")
    evidence = Path("evidence") / ("lifecycle-" + run_id)
    deadline = time.monotonic() + 90
    while time.monotonic() < deadline:
        if active.poll() is not None:
            raise RuntimeError("Trial ended before crash checkpoint")
        forwards = list(evidence.rglob("port-forward.log"))
        if any("Forwarding from 127.0.0.1:18080" in p.read_text() for p in forwards):
            break
        time.sleep(0.2)
    else:
        raise RuntimeError("Crash checkpoint not reached")
    record = json.loads((evidence / "record.json").read_text())
    assert record["state"] == "running" and record["process_group_id"] > 0
    owned_paths = list(evidence.rglob("ownership.json"))
    assert len(owned_paths) == 1
    node = json.loads(owned_paths[0].read_text())
    assert command("docker", "inspect", "--format", "{{.Id}}", node["id"]) == node["id"]
    refusal = subprocess.run([".build/trial-recover", "-run-id", run_id], text=True, capture_output=True, timeout=10)
    assert refusal.returncode != 0 and "active" in refusal.stderr
    (bundle / (name + "-active-refusal.log")).write_text(refusal.stderr)
    # SIGKILL intentionally bypasses the original supervisor's cleanup/defers.
    active.kill()
    active.communicate(timeout=10)
    active = None
    if kill_children:
        identity = command("ps", "-p", str(record["process_group_id"]), "-o", "lstart=", "-o", "command=")
        assert identity == record["process_identity"] and "lifecycle-owner=" + run_id in identity
        os.killpg(record["process_group_id"], signal.SIGKILL)
        time.sleep(0.2)
        assert command("docker", "inspect", "--format", "{{.Id}}", node["id"]) == node["id"]
    assert not (evidence / "lifecycle.json").exists(), "Crash unexpectedly wrote normal completion"
    first_recovery = subprocess.run([".build/trial-recover", "-run-id", run_id], text=True, capture_output=True, timeout=65)
    (bundle / (name + "-recover.log")).write_text(first_recovery.stdout + first_recovery.stderr)
    assert first_recovery.returncode == 0, first_recovery.stdout + first_recovery.stderr
    recovered = json.loads((evidence / "recovery.json").read_text())
    assert recovered["original_outcome"] == "unknown" and recovered["business_verdict"] == "inconclusive"
    assert recovered["processes_verified"] and recovered["cleanup"]["complete"]
    assert not command("docker", "ps", "-aq", "--filter", "label=io.x-k8s.kind.cluster=" + node["cluster"])
    assert not list(evidence.rglob("kubeconfig"))
    again = subprocess.run([".build/trial-recover", "-run-id", run_id], text=True, capture_output=True, timeout=65)
    assert again.returncode == 0, again.stdout + again.stderr
    repeated = json.loads((evidence / "recovery.json").read_text())
    assert repeated["business_verdict"] == "inconclusive" and repeated["original_outcome"] == "unknown"
    assert command("docker", "inspect", "--format", "{{.State.Status}}", sentinel) == "created"
    results[name] = {"run_id": run_id, "recorded_node": node, "recovery": recovered,
                     "active_run_refused": True, "idempotent_recovery": True,
                     "sentinel_preserved": True, "evidence_path": str(evidence)}
    print(name + ": recovered owned resources; outcome unknown; repeat was idempotent", flush=True)


try:
    sentinel = command("docker", "create", "--name", "itl-recovery-sentinel-" + owner[:12],
                       "--label", "incident-trial-lab.recovery-test-owner=" + owner,
                       "kindest/node@sha256:099e049362a1526b2db71494e1947aae99bd16290d7c895f2b7ea312e3cbfaed",
                       "/bin/sleep", "300")
    check_case("supervisor-sigkill", False)
    check_case("supervisor-and-group-sigkill", True)
finally:
    if active is not None and active.poll() is None:
        active.send_signal(signal.SIGTERM)
        active.wait(timeout=80)
    if sentinel is not None:
        labels = json.loads(command("docker", "inspect", "--format", "{{json .Config.Labels}}", sentinel))
        assert labels.get("incident-trial-lab.recovery-test-owner") == owner
        command("docker", "rm", sentinel)
    (bundle / "summary.json").write_text(json.dumps(results, indent=2) + "\n")
print("Recovery evidence: " + str(bundle), flush=True)
