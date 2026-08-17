# Goro website

This directory contains the dependency-free static project website.

Preview it locally from the repository root:

```sh
python3 -m http.server 8080 --directory website
```

Then open <http://localhost:8080>.

The Pages workflow publishes this directory when website changes land on `main`,
or when the workflow is started manually. The current `website` branch is not
published automatically.
