#!/bin/bash

echo "Installing semgrep..."
pip install semgrep --quiet

echo "Running security analysis..."

PYTHONIOENCODING=utf-8 semgrep scan \
  --config=p/security-audit \
  --config=p/owasp-top-ten \
  --config=p/golang \
  --config=p/python \
  --json \
  --output docs/oblak_docs/security_results.json \
  .

echo "Done. Results saved to: docs/oblak_docs/security_results.json"