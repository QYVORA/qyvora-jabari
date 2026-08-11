# Getting Started

## Build

```sh
make build
export PATH="$PWD/bin:$PATH"
jabari version
```

## Interactive console

Running `jabari` with no subcommand drops you into the interactive,
Metasploit-style console:

```sh
jabari
```

You get a colored prompt (`jabari > `), a live status strip showing the
current target and profile, tab completion, command history, and every
one-shot command as a console command:

```text
jabari > target usb
jabari > assess
jabari > report list
jabari > help
jabari > quit
```

The same commands work identically at the shell prompt for one-shot use.

## First assessment

### 1. Connect a device over USB

```sh
adb devices -l
```

You should see one device listed as `device`. If it shows `unauthorized`,
approve the ADB authorization prompt on the device.

### 2. Assess it

```sh
jabari assess usb
```

The framework will:

1. Confirm you intend to assess this device (authorization gate).
2. Run discovery, enumeration, analysis, validation, and risk.
3. Print a terminal report and save a session JSON under `reports/`.

### 3. Review the report

```sh
# list saved sessions
jabari report --list

# re-render the latest session as Markdown
jabari report -f markdown

# as JSON
jabari report -f json
```

### 4. Explore

```sh
jabari target usb          # select a target explicitly
jabari discover            # device metadata
jabari enumerate           # application inventory
jabari analyze             # run the rules
jabari validate            # confirm findings non-destructively
jabari report              # render the session
```

## Network target

To assess a device reachable by ADB over the network (e.g. an authorized
device on your test bench):

```sh
jabari assess ip 192.168.1.50 -y
```

JABARI contacts **only** `192.168.1.50`. It does not probe the rest of the
network. The device must have ADB-over-network enabled (`adb tcpip 5555`).

## Profiles

Choose a profile to control how deep the assessment goes:

```sh
jabari assess usb --profile quick
jabari assess usb --profile deep
jabari assess usb --profile application
```

See [Configuration](configuration.md) for the full list.

## Non-interactive / automation

```sh
jabari assess usb -y --json        # authorized, JSON to stdout
jabari assess ip 10.0.0.5 -y -o reports/day1.json
```

For automated pipelines, set the profile, format, and authorization in the
config file so invocations stay short:

```yaml
profile: standard
output: json
authorized: true
```

> `authorized: true` in a config file is a strong statement. Use it only in
> environments you control; see [Security Model](security-model.md).
