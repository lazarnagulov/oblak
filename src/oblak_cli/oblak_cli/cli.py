import requests
from pathlib import Path
from typing import Optional

import typer
import rich
from rich.table import Table

from . import auth


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
    url = auth.get_server_url()
    rich.print(f"[yellow]Connecting to the server ({url})...[/yellow]")
    
    username = username.strip()
    password = password.strip()
    
    try:
        response = requests.post(
            f"{url}/auth/login",
            json={"username": username, "password": password},
            timeout=5
        )
        
        if response.status_code == 200:
            data = response.json()
            auth.save_token(data["token"], username)
            rich.print("[bold green]Authentication successful.[/bold green]")
            rich.print(f"Welcome back, [bold]{username}[/bold].")
        else:
            error_msg = response.json().get("error", "Unknown error")
            rich.print(f"[bold red]Login failed:[/bold red] {error_msg}")

    except requests.exceptions.ConnectionError:
        rich.print(f"[bold red]Error:[/bold red] Unable to connect to the server at {url}.")
    

@auth_app.command("logout")
def logout():
    if auth.get_token():
        auth.delete_auth_data()
        rich.print("[bold green]Logout successful.[/bold green] Local token has been removed.")
    else:
        rich.print("[yellow]You are already logged out.[/yellow]")

@auth_app.command("whoami")
def whoami():
    username = auth.get_username()
    if not username:
        rich.print("[bold red]You are not logged in.[/bold red] Run `oblak auth login`.")
        return
    rich.print(f"You are logged in as: [bold blue]{username}[/bold blue]")
    

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
    rich.print(f"API URL: [bold]{auth.get_server_url()}[/bold]")

    username = auth.get_username()
    status = f"Authenticated ({username})" if username else "Not authenticated"
    rich.print(f"Auth Status: {status}")

@config_app.command("set-server")
def config_set_server(url: str = typer.Argument(...),):
    auth.set_server_url(url)
    rich.print("[bold green]CONFIG SET SERVER[/bold green]")
    rich.print(f"API URL successfully set to: [bold]{url}[/bold]")

def create_app() -> typer.Typer:
    return app   

