# Oblak CLI

CLI for interacting with the Oblak secure serverless execution platform.
The CLI provides commands for:

- Authentication
- Function deployment and management
- Function invocation
- Execution management
- CLI configuration

Built with [Typer](https://typer.tiangolo.com/) and [Rich](https://rich.readthedocs.io/en/stable/).

## Development Setup
From the repo root:
```bash
cd oblak_cli
```
Create a virtual environment:
```bash
python -m venv .venv
source .venv/bin/activate
```
Install dependencies:
```bash
pip install -r requirements.txt
```
Install the CLI locally:
```bash
pip install -e .
```

## Running the CLI
Show all available commands:
```bash
oblak --help
```
Run directly with Python:
```bash
python -m oblak_cli.main
```
### Authentication Commands
```bash
oblak auth login  
oblak auth logout
oblak auth whoami
```
### Function Commands
```bash
oblak function deploy ./my-function
oblak function deploy ./my-function -r requirements.txt
oblak function deploy ./my-function -n hello-world
oblak function list
oblak function describe hello-world
oblak function delete hello-world
oblak function scan-results hello-world
```
### Function Invocation
```bash
oblak invoke hello-world
oblak invoke hello-world -j '{"message":"hello"}'
oblak invoke hello-world --async
```
### Execution Commands
```bash
oblak execution list
```
### Configuration Commands
```bash
oblak config show
oblak config set-server https://api.oblak.local
```
## Example Workflow
```bash
# Login
oblak auth login

# Deploy a function
oblak function deploy ./examples/hello-world -n hello-world

# List functions
oblak function list

# Invoke function
oblak invoke hello-world -j '{"message":"Hello"}'

# View executions
oblak execution list
```