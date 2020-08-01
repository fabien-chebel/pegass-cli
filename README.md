# Pegass CLI

A command-line interface to the French Red Cross planning web app, Pegass.

## Usage 

This app requires a valid account on Pegass.

Prior running the app, you must create a `config.json` file with the following content
```json
{
  "username": "PEGASS USERNAME GOES HERE",
  "password": "PEGASS PASSWORD GOES HERE"
}
```

This file will allow you to authenticate using the following command: `pegass-cli login`.

Once logged-in, you may run any of the supported commands.