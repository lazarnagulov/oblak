from pathlib import Path
from typing import Optional

import typer
import rich
from rich.table import Table


app = typer.Typer(
    name="oblak",
    help="Oblak CLI - Secure serverless execution platform",
    no_args_is_help=True,
)
auth_app = typer.Typer(help="Authentication commands")
function_app = typer.Typer(help="Function management commands")
execution_app = typer.Typer(help="Execution management commands")
config_app = typer.Typer(help="Configuration commands")

app.add_typer(auth_app, name="auth")
app.add_typer(function_app, name="function")
app.add_typer(execution_app, name="execution")
app.add_typer(config_app, name="config")

@auth_app.command("login")
def login(
    username: str = typer.Option(..., prompt=True),
    password: str = typer.Option(..., prompt=True, hide_input=True),
):
    rich.print("[bold green]AUTH LOGIN[/bold green]")
    rich.print(f"Username: {username}")
    rich.print(f"Password length: {len(password)}")

@auth_app.command("logout")
def logout():
    rich.print("[bold red]AUTH LOGOUT[/bold red]")

@auth_app.command("whoami")
def whoami():
    rich.print("[bold blue]AUTH WHOAMI[/bold blue]")
    rich.print("Current user: demo-user")


@function_app.command("deploy")
def deploy(
    path: Path = typer.Argument(..., exists=True),
    requirements: Optional[Path] = typer.Option(
        None,
        "--requirements",
        "-r",
        exists=True,
        help="Path to requirements.txt",
    ),
    name: Optional[str] = typer.Option(
        None,
        "--name",
        "-n",
        help="Function name",
    ),
):
    rich.print("[bold green]FUNCTION DEPLOY[/bold green]")
    rich.print(f"Path: {path}")

    if requirements:
        rich.print(f"Requirements: {requirements}")

    if name:
        rich.print(f"Function name: {name}")

@function_app.command("list")
def list_functions():
    rich.print("[bold cyan]FUNCTION LIST[/bold cyan]")

    table = Table(title="Functions")

    table.add_column("Name")
    table.add_column("Runtime")
    table.add_column("Status")

    table.add_row("hello-world", "python3.12", "ACTIVE")
    table.add_row("data-parser", "python3.12", "ACTIVE")

    rich.print(table)

@function_app.command("describe")
def describe(function_name: str = typer.Argument(...),):
    rich.print("[bold yellow]FUNCTION DESCRIBE[/bold yellow]")
    rich.print(f"Function: {function_name}")

@function_app.command("delete")
def delete(
    function_name: str = typer.Argument(...),
    force: bool = typer.Option(
        False,
        "--force",
        "-f",
        help="Delete without confirmation",
    ),
):
    rich.print("[bold red]FUNCTION DELETE[/bold red]")
    rich.print(f"Function: {function_name}")
    rich.print(f"Force: {force}")

@function_app.command("scan-results")
def scan_results(function_name: str = typer.Argument(...),):
    rich.print("[bold magenta]FUNCTION SCAN RESULTS[/bold magenta]")
    rich.print(f"Function: {function_name}")

@app.command("invoke")
def invoke(
    function_name: str = typer.Argument(...),
    payload: Optional[str] = typer.Option(
        None,
        "--json",
        "-j",
        help="Inline JSON payload",
    ),
    async_execution: bool = typer.Option(
        False,
        "--async",
        help="Invoke function asynchronously",
    ),
):
    rich.print("[bold green]FUNCTION INVOKE[/bold green]")
    rich.print(f"Function: {function_name}")

    if payload:
        rich.print(f"Payload: {payload}")

    rich.print(f"Async: {async_execution}")

@execution_app.command("list")
def execution_list():
    rich.print("[bold cyan]EXECUTION LIST[/bold cyan]")

    table = Table(title="Executions")

    table.add_column("ID")
    table.add_column("Function")
    table.add_column("Status")

    table.add_row("exec-001", "hello-world", "SUCCESS")
    table.add_row("exec-002", "data-parser", "RUNNING")

    rich.print(table)

@config_app.command("show")
def config_show():
    rich.print("[bold cyan]CONFIG SHOW[/bold cyan]")
    rich.print("API URL: https://api.oblak.local")
    rich.print("Profile: default")

@config_app.command("set-server")
def config_set_server(url: str = typer.Argument(...),):
    rich.print("[bold green]CONFIG SET SERVER[/bold green]")
    rich.print(f"New URL: {url}")

def create_app() -> typer.Typer:
    return app   

