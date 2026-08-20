"""Environment-driven configuration."""

from __future__ import annotations

import pytest

from tenableio_mock.config import OnOmit, settings_from_env


def test_defaults_are_strict() -> None:
    settings = settings_from_env({})
    assert settings.quirks.on_omitted_description is OnOmit.CLEARS
    assert settings.quirks.on_omitted_filters is OnOmit.CLEARS
    assert settings.quirks.lowercase_category_names is False
    assert settings.quirks.reject_unknown_fields is False
    assert settings.access_key == ""
    assert settings.seed is True
    assert settings.frozen_clock is True


def test_quirks_are_read_from_the_environment() -> None:
    settings = settings_from_env(
        {
            "MOCK_OMITTED_DESCRIPTION": "preserves",
            "MOCK_OMITTED_FILTERS": "preserves",
            "MOCK_LOWERCASE_CATEGORY_NAMES": "true",
            "MOCK_REJECT_UNKNOWN_FIELDS": "1",
        }
    )
    assert settings.quirks.on_omitted_description is OnOmit.PRESERVES
    assert settings.quirks.on_omitted_filters is OnOmit.PRESERVES
    assert settings.quirks.lowercase_category_names is True
    assert settings.quirks.reject_unknown_fields is True


@pytest.mark.parametrize("raw", ["1", "true", "TRUE", "yes", "on"])
def test_truthy_spellings(raw: str) -> None:
    assert settings_from_env({"MOCK_LOWERCASE_CATEGORY_NAMES": raw}).quirks.lowercase_category_names


@pytest.mark.parametrize("raw", ["0", "false", "no", "off", "anything-else"])
def test_falsy_spellings(raw: str) -> None:
    assert not settings_from_env(
        {"MOCK_LOWERCASE_CATEGORY_NAMES": raw}
    ).quirks.lowercase_category_names


def test_empty_variable_falls_back_to_the_default() -> None:
    assert settings_from_env({"MOCK_SEED": ""}).seed is True


def test_an_invalid_omit_policy_fails_loudly() -> None:
    """Better a startup crash than a mock silently running the wrong semantics."""
    with pytest.raises(ValueError, match="MOCK_OMITTED_DESCRIPTION"):
        settings_from_env({"MOCK_OMITTED_DESCRIPTION": "sometimes"})


def test_credentials_and_user() -> None:
    settings = settings_from_env(
        {"MOCK_ACCESS_KEY": "a", "MOCK_SECRET_KEY": "s", "MOCK_USER": "me@example.com"}
    )
    assert (settings.access_key, settings.secret_key) == ("a", "s")
    assert settings.user == "me@example.com"


def test_settings_do_not_leak_into_the_ambient_environment() -> None:
    import os

    settings_from_env({"MOCK_USER": "temporary@example.com"})
    assert "MOCK_USER" not in os.environ
