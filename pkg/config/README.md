Config package is here to provide a prioritized configuration registry for all subsystems/packages.  As a separate pkg, this one should be reachable (without circular dependency) by any other node package.  Based on `github.com/spf13/viper` it provides:


- A separate package to expose by value all loaded configs to any node package
     - No cyclic dependency issues
     - No config changes after the initial loading by main pkg

- A prioritized registry for different config sources (flags, config file, env)
- Support for a minimal config language (TOML)
- Support for other config formats like JSON, YAML, HCL
    - *.json, *.toml, *.yaml, *.yml, *.properties, *.props, *.prop, *.hcl
- Live-watching and re-reading of config files
- Unmarshal-to-structs for better code readability and easier maintenance
- config examples - `default.dusk.toml`
- Multiple config file searchPaths
    - current working directory
    - $HOME/.dusk/
- Remote Key/Value Store (not yet in-use)

Example usage:

```bash

# Try to load a config file from any of the searchPaths
# and overwrite general.network setting
user$ ./testnet --general.network=testnet

# Load a specified config file and overwrite logger.level
# config file name can be in form of dusk.toml, dusk.json, dusk.yaml, dusk.tf
user$ ./testnet --config=./pkg/config/default.dusk.toml --logger.level=error

# Load config file found in $searchPaths and overwrite general.network value
user$ DUSK_GENERAL_NETWORK=mainnet; ./testnet

# Load config where a file config value is overwritten by both ENV var and CLI flag but CLI flag has highest priority
user$ DUSK_LOGGER_LEVEL=WARN; ./testnet --logger.level=error

```

More detailed and up-to-date examples about supported flags, env, aliases can be found in `loader_test.go`  

## Viper

```
Viper is a prioritized configuration registry. It maintains a set of configuration sources, fetches values to populate those, and provides them according to the source's priority.

The priority of the sources is the following:
1. overrides
2. flags
3. env. variables
4. config file
5. key/value store
6. defaults
```