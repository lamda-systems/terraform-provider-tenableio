"""Run the mock: ``python -m tenableio_mock [--host H] [--port P]``.

Configuration comes from the environment (see
:func:`tenableio_mock.config.settings_from_env`); the flags here only control
where the server listens.
"""

from __future__ import annotations

import argparse

import uvicorn


def main() -> None:
    parser = argparse.ArgumentParser(
        prog="tenableio_mock",
        description="In-memory Tenable.io API for testing the Terraform provider.",
    )
    parser.add_argument(
        "--host",
        default="127.0.0.1",
        help="interface to bind (default: 127.0.0.1; use 0.0.0.0 in a container)",
    )
    parser.add_argument("--port", type=int, default=8080, help="port to bind (default: 8080)")
    parser.add_argument(
        "--log-level",
        default="info",
        choices=("critical", "error", "warning", "info", "debug", "trace"),
    )
    parser.add_argument(
        "--reload", action="store_true", help="restart on source changes (development)"
    )
    args = parser.parse_args()

    uvicorn.run(
        "tenableio_mock.app:app",
        host=args.host,
        port=args.port,
        log_level=args.log_level,
        reload=args.reload,
    )


if __name__ == "__main__":
    main()
