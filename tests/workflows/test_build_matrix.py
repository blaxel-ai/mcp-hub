"""Exercise the production workflow's matrix script against real Git changes."""

import json
import os
from pathlib import Path
import subprocess
import tempfile
import textwrap
import unittest


WORKFLOW = Path(__file__).resolve().parents[2] / ".github/workflows/build-and-push.yaml"


def matrix_script():
    lines = WORKFLOW.read_text().splitlines()
    step = lines.index("      - name: Retrieve hub servers")
    start = lines.index("        run: |", step) + 1
    end = start
    while end < len(lines) and (not lines[end].strip() or lines[end].startswith("          ")):
        end += 1
    return textwrap.dedent("\n".join(lines[start:end]))


class BuildMatrixTest(unittest.TestCase):
    def select(self, changed, *, mcp="", all_servers="false"):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            for name, extra in {"signoz": "", "slack": "", "disabled": "disabled: true\n", "future": "comingSoon: true\n"}.items():
                path = root / "hub" / f"{name}.yaml"
                path.parent.mkdir(exist_ok=True)
                path.write_text("transport: http-stream\n" + extra)
            for name in ("signoz", "slack", "disabled", "future"):
                path = root / "dockerfiles" / f"{name}.Dockerfile"
                path.parent.mkdir(exist_ok=True)
                path.write_text("FROM alpine\n")

            def git(*args):
                return subprocess.run(
                    ["git", "-c", "user.name=Test", "-c", "user.email=test@example.com", "-c", "commit.gpgsign=false", "-c", "core.hooksPath=/dev/null", *args],
                    cwd=root, check=True, capture_output=True, text=True,
                )

            git("init", "-q")
            git("add", ".")
            git("commit", "-qm", "Baseline")
            for name in changed:
                path = root / name
                path.parent.mkdir(parents=True, exist_ok=True)
                with path.open("a") as file:
                    file.write("# changed\n")
            git("add", ".")
            git("commit", "--allow-empty", "-qm", "Change")
            output = root / "output"
            script = matrix_script().replace("${{ inputs.mcp }}", mcp).replace("${{ inputs.all }}", all_servers)
            result = subprocess.run(
                ["bash", "-c", script], cwd=root,
                env={**os.environ, "GITHUB_OUTPUT": str(output)},
                capture_output=True, text=True,
            )
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertTrue(output.exists(), result.stdout)
            return json.loads(output.read_text().removeprefix("servers="))

    def test_one_dockerfile_selects_only_its_server(self):
        self.assertEqual(self.select(["dockerfiles/signoz.Dockerfile"]), ["signoz"])

    def test_multiple_dockerfiles_select_their_servers(self):
        self.assertEqual(self.select(["dockerfiles/signoz.Dockerfile", "dockerfiles/slack.Dockerfile"]), ["signoz", "slack"])

    def test_shared_code_selects_all_active_servers(self):
        for path in ("super-gateway/main.go", "internal/docker/inject.go", "cmd/import.go"):
            with self.subTest(path=path):
                self.assertEqual(self.select([path]), ["signoz", "slack"])

    def test_hub_config_selects_only_its_server(self):
        self.assertEqual(self.select(["hub/slack.yaml"]), ["slack"])

    def test_disabled_and_coming_soon_servers_are_excluded(self):
        self.assertEqual(self.select(["dockerfiles/disabled.Dockerfile", "dockerfiles/future.Dockerfile"]), [])

    def test_unrelated_change_selects_no_servers(self):
        self.assertEqual(self.select(["README.md"]), [])

    def test_dispatch_all_selects_active_servers(self):
        self.assertEqual(self.select([], all_servers="true"), ["signoz", "slack"])

    def test_dispatch_one_selects_requested_server(self):
        self.assertEqual(self.select(["super-gateway/main.go"], mcp="signoz"), ["signoz"])


if __name__ == "__main__":
    unittest.main()
