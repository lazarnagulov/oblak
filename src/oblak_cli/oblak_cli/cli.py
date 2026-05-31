import json
import tomllib

import requests
from pathlib import Path
from typing import Optional

import typer
import rich
from rich.table import Table

from . import auth
from . import manifest as oblak_manifest

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
    path: Path = typer.Argument(..., exists=True, help="Python file or project directory"),
    manifest: Optional[Path] = typer.Option(None, "--manifest", "-m", exists=True),
    requirements: Optional[Path] = typer.Option(None, "--requirements", "-r", exists=True),
    name: Optional[str] = typer.Option(None, "--name", "-n"),
    handler: str = typer.Option("handler", "--handler", "-h"),
    runtime: str = typer.Option("python3.12", "--runtime"),
    timeout: int = typer.Option(5, "--timeout", min=1, max=30),
    memory: int = typer.Option(128, "--memory", min=64, max=512),
):
    rich.print(f"[yellow]Preparing deployment for {path.name}...[/yellow]")

    try:
        manifest_data = oblak_manifest.get_or_create_manifest(path, name, manifest, runtime, handler, timeout, memory)
        zip_buffer = oblak_manifest.build_zip_artifact(path, manifest_data, requirements)
    except ValueError as e:
        rich.print(f"[bold red]Packaging Error:[/bold red] {e}")
        raise typer.Exit(1)

    form_data = {"manifest": json.dumps(manifest_data)}
    files = {"artifact": (f"{manifest_data['name']}.zip", zip_buffer, "application/zip")}
    
    try:
        response = requests.post(
            f"{auth.get_server_url()}/functions", 
            headers=auth.get_auth_headers(), 
            data=form_data, 
            files=files,
            timeout=30
        )
        
        if response.status_code in (200, 201):
            rich.print("[bold green]Deployment successful![/bold green]")
            rich.print(f"Your function is now available at: [bold blue]{response.json().get('access_url')}[/bold blue]")
        else:
            error_msg = response.json().get("error", response.text)
            rich.print(f"[bold red]Deploy failed:[/bold red] {error_msg}")
            
    except requests.exceptions.RequestException as e:
        rich.print(f"[bold red]Network Error:[/bold red] {e}")    
    
@function_app.command("list")
def list_functions():
    rich.print("[bold cyan]Fetching your functions...[/bold cyan]")
    
    try:
        response = requests.get(
            f"{auth.get_server_url()}/functions", 
            headers=auth.get_auth_headers(), 
            timeout=10
        )
        response.raise_for_status()
        
        data = response.json()
        funcs = data.get("functions", [])
        
        if not funcs:
            rich.print("[yellow]You have no deployed functions.[/yellow]")
            return

        table = Table(title="Your Functions")
        table.add_column("Name", style="cyan")
        table.add_column("Runtime", style="green")

        for f in funcs:
            table.add_row(f.get("name"), f.get("runtime"))

        rich.print(table)
        
    except Exception as e:
        rich.print(f"[bold red]Error:[/bold red] {e}")
        
@function_app.command("describe")
def describe(function_name: str = typer.Argument(..., help="Name of the function")):
    rich.print(f"[bold yellow]Fetching details for '{function_name}'...[/bold yellow]")
    
    try:
        response = requests.get(
            f"{auth.get_server_url()}/functions/{function_name}", 
            headers=auth.get_auth_headers(), 
            timeout=10
        )
        response.raise_for_status()
        
        f = response.json()
        
        table = Table(show_header=False)

        table.add_column("Field", style="bold cyan")
        table.add_column("Value")
                
        table.add_row("Name", str(f.get("name")))
        table.add_row("Runtime", str(f.get("runtime")))
        table.add_row("Module", str(f.get("module")))
        table.add_row("Handler", str(f.get("handler")))
        table.add_row("Memory", f"{f.get('memory')}MB")
        table.add_row("Timeout", f"{f.get('timeout')}s")
        table.add_row("Created at", str(f.get("created_at")))
        
        rich.print(table)
    except requests.exceptions.HTTPError:
        rich.print(f"[bold red]Error:[/bold red] Function '{function_name}' not found.")
    except Exception as e:
        rich.print(f"[bold red]Error:[/bold red] {e}")

@function_app.command("delete")
def delete(
    function_name: str = typer.Argument(..., help="Name of the function"),
):
    rich.print(f"[yellow]Deleting '{function_name}'...[/yellow]")

    try:
        response = requests.delete(
            f"{auth.get_server_url()}/functions/{function_name}",
            headers=auth.get_auth_headers(),
            timeout=10,
        )
        response.raise_for_status()
        rich.print("[bold green]Delete successful.[/bold green]")
    except requests.exceptions.HTTPError:
        if response.status_code == 404:
            rich.print(
                f"[bold red]Delete failed:[/bold red] "
                f"Function '{function_name}' not found."
            )
        else:
            rich.print(
                f"[bold red]Delete failed:[/bold red] "
                f"{response.text}"
            )
    except requests.exceptions.RequestException as e:
        rich.print(f"[bold red]Network Error:[/bold red] {e}")
    
@function_app.command("scan-results")
def scan_results(function_name: str = typer.Argument(...),):
    rich.print("[bold magenta]FUNCTION SCAN RESULTS[/bold magenta]")
    rich.print(f"Function: {function_name}")

@function_app.command("generate-url")
def generate_link(function_name: str = typer.Argument(...),):
    rich.print(f"[bold yellow]Generating new access URL for '{function_name}'...[/bold yellow]")

    try:
        response = requests.get(
            f"{auth.get_server_url()}/functions/{function_name}/generate-url",
            headers=auth.get_auth_headers(),
            timeout=10,
        )
        response.raise_for_status()
        access_url = response.json().get("access_url")
        rich.print("[bold green]New access URL generated:[/bold green]")
        rich.print(f"[bold blue]{access_url}[/bold blue]")

    except requests.exceptions.HTTPError:
        rich.print(f"[bold red]Error:[/bold red] Failed to generate access URL for function '{function_name}'.")
    except Exception as e:
        rich.print(f"[bold red]Error:[/bold red] {e}")

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

