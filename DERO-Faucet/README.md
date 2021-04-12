# DERO Faucet
[DERO Faucet](https://derofaucet.nelbert442.com)

For testnet purposes: I recommend generating server wallet to host the service. Just to confirm no other services/dApps have used the ports defined (though not likely)

### Disclaimer
This implementation is under heavy development and this version is only for testing purposes at this time.

### Server Service (owner of userKey gen, API, FrontEnd)
DeroFaucet-Server.go contains the server service code. This can be ran locally, or on a server intended for hosting the API/Frontend publicly for user consumption.

DeroFaucet-Server usage and help output below (heavy development and will be modified in future iterations)
```
DeroFaucet-Server
DERO Faucet Service (server): A 2 way service implementation for interacting with a Smart Contract while ensuring input variables are as private as possible

Usage:
  DeroFaucet-Server [options]
  DeroFaucet-Server -h | --help

Options:
  -h --help     Show this screen.
  --daemon-address=<host:port>	Use daemon instance at <host>:<port> or https://domain
  --rpc-server-address=<127.0.0.1:40403>	connect to service (server) wallet
  --api-address=<0.0.0.0:8224>	API (non-SSL) will be enabled at the defined address (or defaulted to 0.0.0.0:8224)
  --ssl-api-address=<0.0.0.0:8225>	if defined, API (SSL) will be enabled at the defined address. apifullchain.cer && apicert.key in the same dir is required
  --frontend-port=<8080>	if defined, frontend (non-SSL) will be enabled
  --ssl-frontend-port=<8181>	if defined, frontend (SSL) will be enabled. fefullchain.cer && fecert.key in the same dir is required
```

### Backend DB for Server/Client Service (graviton)
In usual form with DERO projects I have taken on recently, I leverage [Graviton](https://github.com/deroproject/graviton) for the backend DB store.