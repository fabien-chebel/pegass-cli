# Pegass CLI

A command-line interface to the French Red Cross planning web app, Pegass.

## Usage 

This app requires a valid account on Pegass.

Prior to running the app, you must create a `config.json` file with the following content
```json
{
  "username": "PEGASS USERNAME GOES HERE",
  "password": "PEGASS PASSWORD GOES HERE",
  "totp_secret_key": "TOTP SECRET KEY GOES HERE"
}
```

This file will allow you to authenticate using the following command: `pegass-cli login`.

Once logged-in, you may run any of the supported commands.

```
NAME:
   Pegass CLI - Interact with Red Cross's Pegass web app through the CLI

USAGE:
   pegass-cli [global options] command [command options] [arguments...]

COMMANDS:
   login    Authenticate to Pegass
   whoami   Get current user information
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --help, -h  show help
```