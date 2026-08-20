VENV=.venv
PY=$(VENV)/bin/python

.PHONY: venv test run lint clean docker-build

venv:
	python3 -m venv $(VENV)
	$(PY) -m pip install --upgrade pip
	$(PY) -m pip install -r requirements-dev.txt

test:
	$(PY) -m pytest

# Serve on all interfaces so the provider can reach it from another container.
run:
	$(PY) -m tenableio_mock --host 0.0.0.0 --port 8080

# Reproduce the apply failure that motivated the mock.
run-quirky:
	MOCK_OMITTED_DESCRIPTION=preserves \
	MOCK_LOWERCASE_CATEGORY_NAMES=1 \
	$(PY) -m tenableio_mock --host 0.0.0.0 --port 8080

docker-build:
	docker build -t tenableio-mock:local .

clean:
	rm -rf $(VENV) .pytest_cache
	find . -type d -name __pycache__ -prune -exec rm -rf {} +
