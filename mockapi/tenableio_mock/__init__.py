"""An in-memory fake of the Tenable.io REST API.

Covers every endpoint ``internal/client`` calls. The point is fidelity, not
convenience: it echoes back exactly what it was given, rejects what the real
API rejects, and reproduces the response shapes that differ from the request
shapes. A mock more forgiving than production is worse than none, because it
certifies provider bugs as passing.

Where the docs leave a behaviour genuinely ambiguous the mock does not guess.
It implements the conservative reading by default and offers the alternative as
a quirk, so the provider can be proven correct either way. See
:class:`tenableio_mock.config.Quirks`.
"""

from .app import create_app
from .config import OnOmit, Quirks, Settings, settings_from_env

__all__ = ["create_app", "OnOmit", "Quirks", "Settings", "settings_from_env"]
