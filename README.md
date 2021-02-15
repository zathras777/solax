# solax
Solax inverter to MQTT daemon

## Command Line Options
```
Usage of ./solax:
  -cfg string
        Configuration file to parse for database settings (default "configuration.yaml")
  -daemon
        Run as daemon? [default false]
```
When run without the -daemon flag the app will simply read all available values and print them to stdout.

## Configuration
Configuration is done via a YAML file with a default name of configuration.yaml. 

## HomeAssistant Discovery
When run in daemon mode, configuration will be published for HA to automatically configure the available sensors.

This has been running well for me for a while now and should be stable. Suggestions and improvements always welcome.
