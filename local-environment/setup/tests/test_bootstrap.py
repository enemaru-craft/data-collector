"""Bootstrap regression checks without Docker or an AWS connection."""
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest


FAKE_AWS = r'''#!/usr/bin/env python3
import json
import os
from pathlib import Path
import sys

args = sys.argv[1:]
args = args[4:]  # --endpoint-url URL --region REGION
state_path = Path(os.environ["FAKE_STATE"])
state = json.loads(state_path.read_text()) if state_path.exists() else {
    "functions": {}, "urls": [], "table": False, "calls": []}
state["calls"].append(args)
service, operation = args[:2]
name = args[args.index("--function-name") + 1] if "--function-name" in args else None
status = 0
output = "{}"
if service == "dynamodb":
    if operation == "describe-table" and not state["table"]:
        status = 1
    if operation == "create-table":
        state["table"] = True
elif service == "lambda":
    if operation == "get-function" and name not in state["functions"]:
        status = 1
    elif operation in ("create-function", "update-function-code"):
        state["functions"][name] = False
    elif operation == "wait":
        state["functions"][name] = True
    elif operation == "update-function-configuration":
        if not state["functions"].get(name) or "--architectures" in args:
            status = 17
        elif os.environ.get("FAIL_CONFIG") == "1":
            status = 18
        else:
            state["functions"][name] = False
    elif operation == "get-function-url-config":
        if name not in state["urls"]:
            status = 1
        else:
            output = "http://test.lambda-url.ap-northeast-1.localhost.localstack.cloud:4566/"
    elif operation == "create-function-url-config":
        state["urls"].append(name)
    elif operation == "add-permission" and os.environ.get("FAIL_PERMISSION") == "1":
        status = 19
state_path.write_text(json.dumps(state))
if status == 0:
    print(output)
sys.exit(status)
'''


class BootstrapTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        scripts = self.root / "local-environment/scripts"
        scripts.mkdir(parents=True)
        (self.root / "local-environment/nginx").mkdir()
        source = Path(__file__).resolve().parents[2] / "scripts/bootstrap_localstack.sh"
        self.script = scripts / source.name
        shutil.copyfile(source, self.script)
        for module in ("management-device-and-world-data-lambda", "power-data-registration-lambda"):
            zip_path = self.root / "lambda" / module / "lambda_function.zip"
            zip_path.parent.mkdir(parents=True)
            zip_path.write_bytes(b"test fixture")
        bin_dir = self.root / "bin"
        bin_dir.mkdir()
        aws = bin_dir / "aws"
        aws.write_text(FAKE_AWS)
        aws.chmod(0o755)
        self.state_path = self.root / "state.json"
        self.env = dict(os.environ, PATH=f"{bin_dir}:{os.environ['PATH']}",
                        FAKE_STATE=str(self.state_path), SKIP_BUILD="1", SKIP_NGINX_RELOAD="1")

    def run_bootstrap(self, **env):
        return subprocess.run(["bash", str(self.script)], env=dict(self.env, **env),
                              capture_output=True, text=True, timeout=30)

    def test_prebuilt_create_and_update_wait_for_configuration(self):
        for _ in range(2):
            result = self.run_bootstrap()
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertIn("Using prebuilt Lambda ZIPs", result.stdout)
        state = json.loads(self.state_path.read_text())
        self.assertEqual(len(state["functions"]), 2)
        self.assertTrue(all(state["functions"].values()))
        self.assertTrue(any(call[:2] == ["lambda", "update-function-code"] for call in state["calls"]))
        config = self.root / "local-environment/nginx/default.conf"
        self.assertIn("proxy_set_header Host test.lambda-url", config.read_text())

    def test_configuration_failure_is_not_reported_as_success(self):
        result = self.run_bootstrap(FAIL_CONFIG="1")
        self.assertNotEqual(result.returncode, 0)
        self.assertNotIn("env configured", result.stdout)
        self.assertNotIn("[DONE]", result.stdout)

    def test_missing_zip_fails_before_creating_resources(self):
        (self.root / "lambda/power-data-registration-lambda/lambda_function.zip").unlink()
        result = self.run_bootstrap()
        self.assertNotEqual(result.returncode, 0)
        state = json.loads(self.state_path.read_text())
        self.assertFalse(state["table"])
        self.assertEqual(state["functions"], {})

    def test_url_permission_failure_is_not_reported_as_success(self):
        result = self.run_bootstrap(FAIL_PERMISSION="1")
        self.assertNotEqual(result.returncode, 0)
        self.assertNotIn("[DONE]", result.stdout)

    def test_arm64_is_passed_to_lambda_creation(self):
        result = self.run_bootstrap(LAMBDA_ARCHITECTURE="arm64")
        self.assertEqual(result.returncode, 0, result.stderr)
        state = json.loads(self.state_path.read_text())
        creates = [call for call in state["calls"] if call[:2] == ["lambda", "create-function"]]
        self.assertEqual(len(creates), 2)
        for call in creates:
            self.assertEqual(call[call.index("--architectures") + 1], "arm64")

    def test_database_connection_failure_does_not_trigger_creation(self):
        source = Path(__file__).resolve().parents[2] / "scripts/migrate_postgres.sh"
        script = self.script.parent / source.name
        shutil.copyfile(source, script)
        bin_dir = self.root / "bin"
        for command, body in (("pg_isready", "exit 0"), ("psql", "exit 2")):
            stub = bin_dir / command
            stub.write_text("#!/bin/sh\n" + body + "\n")
            stub.chmod(0o755)
        result = subprocess.run(["bash", str(script)], env=dict(self.env, SETUP_CONTAINER="1"),
                                capture_output=True, text=True, timeout=30)
        self.assertNotEqual(result.returncode, 0)
        self.assertNotIn("Creating database", result.stdout)
        self.assertNotIn("マイグレーション完了", result.stdout)


if __name__ == "__main__":
    unittest.main()
