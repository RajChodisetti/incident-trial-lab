#!/usr/bin/env python3
"""Real local interruption checks; creates only a labeled, stopped test sentinel."""
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
validation_id = uuid.uuid4().hex
bundle = Path("evidence") / ("lifecycle-validation-" + datetime.datetime.now(datetime.timezone.utc).strftime("%Y%m%dT%H%M%SZ"))
bundle.mkdir(parents=True, exist_ok=False)
owner_label = "incident-trial-lab.test-owner=" + validation_id
sentinel = None
active = None
results = {}


def command(*args):
    return subprocess.check_output(args, text=True, timeout=20).strip()


def check_group_gone(pgid):
    for _ in range(20):
        try:
            os.killpg(pgid, 0)
        except ProcessLookupError:
            return True
        time.sleep(0.1)
    return False


def run_case(name, args, expected, cancel_at_forward=False, inject=False):
    global active
    env = dict(os.environ)
    env.pop("TRIAL_TEST_FAIL_AFTER_APPLY", None)
    if inject:
        env["TRIAL_TEST_FAIL_AFTER_APPLY"] = "1"
    active = subprocess.Popen([".build/trial-run", *args], env=env,
                              stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
    first = active.stdout.readline()
    if not first.startswith("Lifecycle run "):
        raise RuntimeError("Supervisor did not start: " + first)
    run_id = first.split()[2].rstrip(";")
    evidence = Path("evidence") / ("lifecycle-" + run_id)
    if cancel_at_forward:
        deadline = time.monotonic() + 90
        while time.monotonic() < deadline:
            if active.poll() is not None:
                raise RuntimeError("Trial ended before interruption point")
            forwards = list(evidence.rglob("port-forward.log"))
            if any("Forwarding from 127.0.0.1:18080" in p.read_text() for p in forwards):
                active.send_signal(signal.SIGTERM)
                break
            time.sleep(0.2)
        else:
            active.send_signal(signal.SIGTERM)
            raise RuntimeError("Port-forward interruption point not reached")
    output, _ = active.communicate(timeout=80)
    exit_code = active.returncode
    active = None
    (bundle / (name + ".log")).write_text(first + output)
    report = json.loads((evidence / "lifecycle.json").read_text())
    assert report["outcome"] == expected, report
    assert report["cleanup"]["complete"], report
    assert (exit_code == 0) == (expected == "completed"), report
    assert not list(evidence.rglob("kubeconfig")), "Owned kubeconfig retained"
    assert check_group_gone(report["process_group_id"]), "Owned process group remains"
    for cluster in report["cleanup"]["clusters"]:
        assert not command("docker", "ps", "-aq", "--filter",
                           "label=io.x-k8s.kind.cluster=" + cluster["name"])
    assert command("docker", "inspect", "--format", "{{.Id}}", sentinel) == sentinel
    assert command("docker", "inspect", "--format", "{{.State.Status}}", sentinel) == "created"
    results[name] = {"report": report, "evidence_path": str(evidence),
                     "process_group_gone": True, "sentinel_preserved": True}
    print(name + ": " + report["outcome"] + "; cleanup complete; sentinel preserved", flush=True)


try:
    sentinel = command("docker", "create", "--name", "itl-lifecycle-sentinel-" + validation_id[:12],
                       "--label", owner_label,
                       "kindest/node@sha256:099e049362a1526b2db71494e1947aae99bd16290d7c895f2b7ea312e3cbfaed",
                       "/bin/sleep", "300")
    run_case("timeout-during-create", ["-mode", "baseline", "-timeout", "8s"], "timed_out")
    run_case("term-with-port-forward", ["-mode", "candidate", "-variant", "fixed", "-timeout", "90s"],
             "canceled", cancel_at_forward=True)
    run_case("failure-after-apply", ["-mode", "candidate", "-variant", "fixed", "-timeout", "90s"],
             "failed", inject=True)
    run_case("successful-fixed-trial", ["-mode", "candidate", "-variant", "fixed", "-timeout", "90s"],
             "completed")
finally:
    if active is not None and active.poll() is None:
        active.send_signal(signal.SIGTERM)
        active.wait(timeout=80)
    if sentinel is not None:
        labels = json.loads(command("docker", "inspect", "--format", "{{json .Config.Labels}}", sentinel))
        if labels.get("incident-trial-lab.test-owner") != validation_id:
            raise RuntimeError("Sentinel ownership changed; refusing deletion")
        command("docker", "rm", sentinel)  # Exact ID created by this harness; it was never started.
    (bundle / "summary.json").write_text(json.dumps(results, indent=2) + "\n")
print("Validation evidence: " + str(bundle), flush=True)
