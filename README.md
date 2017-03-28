# go-healthz

Package that implements a healthz healthcheck server.

Inspired by kelseyhightower:
- https://github.com/kelseyhightower/app-healthz
- https://vimeo.com/173610242

Add a `Healthz()` function to your application components, and then register them with this package along with a type and description by adding them to `config.Providers`. This package creates a HTTP server that runs all the registered handlers when hitting `/healthz` and returns any errors.

This server does not use TLS, as most of our applications already run their own TLS servers. One approach is to not expose this server's port directly (except for exposing the basic handler for Kube's liveness and readiness), and rather call the healthz handler internally from a TLS handler that you add to your main TLS server.

## Example usage

### Setup and calling

```golang
logger = logrus.WithField("component", "healthz")
config = healthz.Config{
    BindPort: 80,
    BindAddr: "localhost",
    Hostname: "pod1",
    Providers: []healthz.ProviderInfo{
        {
            Type:        "DBConn",
            Description: "Ensure the database is reachable",
            Check: db, // `db` implements `healthz.HealthCheckable()`
        },
        {
            Type:        "Metric",
            Description: "Watch a key metric for failure",
            Check: service, // `service` implements `healthz.HealthCheckable()`
        },
    },
    Log: logger, // logrus
    ServerErrorLog: log.New(logger.Logger.Writer(), "", 0), // stdlib log helper that sends http.Server errors to logrus.
}
healthServer, err := healthz.New(healthzConfig)
if err != nil {
    return err
}
go healthServer.StartHealthz()
```

## Sensu

Can be used as a Sensu check with the [http plugin](https://github.com/sensu-plugins/sensu-plugins-http), for example:

    "command": "/path/to/ruby /path/to/check-http.rb
      -c /path/to/client/cert/cert.pem
      -q '\"Errors\":null'
      -C /path/to/ca.crt
      -u https://my.service.com/healthz"

Sensu alerts when the check either returns non-200 response, or does not contain `"Errors": null` in the body. This is useful as it communicates the error message and nature of the failure to the person reading the check result.
