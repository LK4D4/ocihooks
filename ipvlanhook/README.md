# ipvlanhook

Hook for setting ipvlan interfaces inside
[runc](https://github.com/opencontainers/runc) containers.

## Install

`go get github.com/LK4D4/ocihooks/ipvlanhook`

## Usage
```json
	"hooks": {
		"prestart": [
			{
				"path": "/usr/local/bin/ipvlanhook",
				"args": [
					"ipvlanhook",
					"-parent=wlp3s0",
					"-address=192.168.100.10/24",
					"-mode=l2"
				]
			}
		]
	},
```
