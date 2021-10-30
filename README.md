# tsk

`tsk` is a quick and magical way to connect your Kubernetes cluster to your
Tailscale Tailnet.

## Installation

`tsk` requires you have Pulumi installed.

```bash
$ brew install pulumi
$ go install github.com/adamgoose/tsk@latest
```

## Configuration

Currently, only CLI flag and Environment Variable configuration is supported,
however file-based configuration is coming soon. For now, try the following.

```bash
# Copy the example .envrc file
cp .envrc.example .envrc

# Edit your .envrc accordingly
vim .envrc

# If you have direnv installed...
direnv allow

# ...otherwise
source .envrc
```

## Usage

Simply run `tsk up`!

Now, you can access in-cluster services with the following DNS name pattern:

```
<service_name>.<namespace>.tsk
```

When you're ready to shut everything down, just run `tsk down`.
